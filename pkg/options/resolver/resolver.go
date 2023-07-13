package resolver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/graph"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/shell"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/log/terminal"
	"github.com/pkg/errors"
)

const (
	rootID = "root"
)

var variableExpression = regexp.MustCompile(`(?m)\$\{?([A-Z0-9_]+)(:(-|\+)([^\}]+))?\}?`)

func ResolveDefaultValue(val string, resolvedOptions map[string]string) string {
	return variableExpression.ReplaceAllStringFunc(val, func(s string) string {
		submatch := variableExpression.FindStringSubmatch(s)
		optionVal, ok := resolvedOptions[submatch[1]]
		if ok {
			return optionVal
		}

		return s
	})
}

// replace all value in the map with the resolved default value
func ResolveDefaultValues(vals map[string]string, resolvedOptions map[string]string) map[string]string {
	ret := make(map[string]string)
	for k, v := range vals {
		resolvedValue := ResolveDefaultValue(v, resolvedOptions)
		if resolvedValue == "" {
			continue
		}

		ret[k] = resolvedValue
	}
	return ret
}

type resolver struct {
	userOptions  map[string]string
	extraValues  map[string]string
	graph        *graph.Graph
	optionValues map[string]config.OptionValue
	log          log.Logger

	// options
	resolveLocal  bool
	resolveGlobal bool
	skipRequired  bool
}

type Option func(r *resolver)

func New(userOptions map[string]string, extraValues map[string]string, logger log.Logger, opts ...Option) *resolver {
	if userOptions == nil {
		userOptions = map[string]string{}
	}

	resolver := &resolver{
		userOptions: userOptions,
		extraValues: extraValues,
		log:         logger,
	}

	for _, o := range opts {
		o(resolver)
	}

	return resolver
}

func WithResolveLocal() Option {
	return func(r *resolver) {
		r.resolveLocal = true
	}
}

func WithResolveGlobal() Option {
	return func(r *resolver) {
		r.resolveGlobal = true
	}
}

func WithSkipRequired(skip bool) Option {
	return func(r *resolver) {
		r.skipRequired = skip
	}
}

func (r *resolver) Resolve(ctx context.Context, options map[string]*types.Option, optionValues map[string]config.OptionValue) (map[string]config.OptionValue, config.DynamicOptions, error) {
	if options == nil {
		options = config.DynamicOptions{}
	}

	// create a new graph
	g := graph.NewGraphOf(graph.NewNode(rootID, nil), "provider option")
	err := addOptionsToGraph(g, options)
	if err != nil {
		return nil, nil, err
	}

	// next add the dependencies
	err = addDependencies(g, options, optionValues)
	if err != nil {
		return nil, nil, err
	}
	r.graph = g
	r.optionValues = optionValues

	// resolve options
	resolvedOptions, dynamicOptions, err := r.resolveOptions(ctx)
	if err != nil {
		return nil, nil, err
	}

	return resolvedOptions, dynamicOptions, nil
}

func (r *resolver) resolveOptions(
	ctx context.Context,
) (map[string]config.OptionValue, config.DynamicOptions, error) {
	// copy options
	resolvedOptions := map[string]config.OptionValue{}
	for optionName, v := range r.optionValues {
		resolvedOptions[optionName] = v
	}

	// resolve options order
	clonedGraph := r.graph.Clone()
	orderedOptions := []string{}
	nextLeaf := clonedGraph.GetNextLeaf(clonedGraph.Root)
	for nextLeaf != clonedGraph.Root {
		orderedOptions = append(orderedOptions, nextLeaf.ID)
		err := clonedGraph.RemoveNode(nextLeaf.ID)
		if err != nil {
			return nil, nil, err
		}

		nextLeaf = clonedGraph.GetNextLeaf(clonedGraph.Root)
	}

	dynamicOptions := config.DynamicOptions{}
	// resolve options in reverse order to walk from highest to lowest
	excludedOptions := map[string]bool{}
	for i := len(orderedOptions) - 1; i >= 0; i-- {
		optionName := orderedOptions[i]
		if excludedOptions[optionName] {
			continue
		}

		newDynOpts, err := r.resolveOption(ctx, optionName, resolvedOptions, excludedOptions)
		if err != nil {
			return nil, nil, errors.Wrap(err, "resolve option "+optionName)
		}
		for k, v := range newDynOpts {
			dynamicOptions[k] = v
		}
	}

	return resolvedOptions, dynamicOptions, nil
}

func (r *resolver) resolveOption(
	ctx context.Context,
	optionName string,
	resolvedOptions map[string]config.OptionValue,
	excludedOptions map[string]bool,
) (config.DynamicOptions, error) {
	dynamicOptions := config.DynamicOptions{}
	node := r.graph.Nodes[optionName]
	option := node.Data.(*types.Option)

	// check if user value exists
	userValue, userValueOk := r.userOptions[optionName]

	// find out options we need to resolve
	if !userValueOk {
		// make sure required is always resolved
		if !option.Required {
			// skip if global
			if !r.resolveGlobal && option.Global {
				return dynamicOptions, nil
			} else if !r.resolveLocal && option.Local {
				return dynamicOptions, nil
			}
		}

		// check if value is already filled
		val, ok := resolvedOptions[optionName]
		if ok {
			if val.UserProvided || option.Cache == "" {
				return dynamicOptions, nil
			} else if option.Cache != "" {
				duration, err := time.ParseDuration(option.Cache)
				if err != nil {
					return nil, errors.Wrapf(err, "parse cache duration of option %s", optionName)
				}

				// has value expired?
				if val.Filled != nil && val.Filled.Add(duration).After(time.Now()) {
					return dynamicOptions, nil
				}
			}
		}
	}

	beforeValue := resolvedOptions[optionName].Value
	beforeChildren := resolvedOptions[optionName].Children

	// resolve option
	if userValueOk {
		resolvedOptions[optionName] = config.OptionValue{
			Value:        userValue,
			UserProvided: true,
		}
	} else if option.Default != "" {
		resolvedOptions[optionName] = config.OptionValue{
			Value: ResolveDefaultValue(option.Default, combine(resolvedOptions, r.extraValues)),
		}
	} else if option.Command != "" {
		optionValue, err := resolveFromCommand(ctx, option, resolvedOptions, r.extraValues)
		if err != nil {
			return nil, err
		}

		resolvedOptions[optionName] = optionValue
	} else {
		resolvedOptions[optionName] = config.OptionValue{}
	}

	// Preserve children
	opt := resolvedOptions[optionName]
	opt.Children = beforeChildren
	resolvedOptions[optionName] = opt

	// is required?
	if !userValueOk && option.Required && resolvedOptions[optionName].Value == "" && !resolvedOptions[optionName].UserProvided {
		if r.skipRequired {
			delete(resolvedOptions, optionName)
			excludeChildren(r.graph.Nodes[optionName], excludedOptions)
			return dynamicOptions, nil
		}

		// check if we can ask a question
		if !terminal.IsTerminalIn {
			return dynamicOptions, fmt.Errorf("option %s is required, but no value provided", optionName)
		}

		r.log.Info(option.Description)
		answer, err := r.log.Question(&survey.QuestionOptions{
			Question:               fmt.Sprintf("Please enter a value for %s", optionName),
			Options:                option.Enum,
			ValidationRegexPattern: option.ValidationPattern,
			ValidationMessage:      option.ValidationMessage,
			IsPassword:             option.Password,
		})
		if err != nil {
			return dynamicOptions, err
		}

		resolvedOptions[optionName] = config.OptionValue{
			Value:        answer,
			UserProvided: true,
		}
	}

	if beforeValue != resolvedOptions[optionName].Value {
		// Fetch dynamic options if necessary
		if option.SubOptionsCommand != "" {
			updatedOpt, newOpts, err := resolveSubOptions(ctx, option, optionName, resolvedOptions, r.extraValues)
			if err != nil {
				return nil, err
			}
			resolvedOptions[optionName] = updatedOpt
			dynamicOptions = newOpts
		}

		children := resolvedOptions[optionName].Children
		// remove children from graph
		if len(children) > 0 {
			for _, childID := range children {
				invalidateSubGraph(r.graph, childID, func(id string) {
					delete(resolvedOptions, id)
					delete(r.userOptions, id)
					excludedOptions[id] = true
				})
			}
		}

		// resolve children again
		for _, child := range node.Childs {
			// check if value is already there
			optionValue, ok := resolvedOptions[child.ID]
			if ok && !optionValue.UserProvided {
				// recompute children
				delete(resolvedOptions, child.ID)
			}
		}
	} else {
		// Rebuild dynamic options from children
		resolvedChildren := resolvedOptions[optionName].Children
		if len(resolvedChildren) > 0 {
			for _, child := range node.Childs {
				childOpt, ok := child.Data.(*types.Option)
				if !ok || !contains(resolvedChildren, child.ID) {
					continue
				}
				dynamicOptions[child.ID] = childOpt
			}
		}
	}

	return dynamicOptions, nil
}

func invalidateSubGraph(g *graph.Graph, id string, afterInvalidation func(id string)) {
	node, ok := g.Nodes[id]
	if node == nil || !ok {
		return
	}
	// collect ids of children before removing the node as it mutates the graph,
	// thus re-ordering the children in the nodes slice which then invalidates the pointer of the current iteration
	children := []string{}
	for _, child := range node.Childs {
		children = append(children, child.ID)
	}

	for _, childID := range children {
		invalidateSubGraph(g, childID, afterInvalidation)
	}
	err := g.RemoveNode(id)
	if err != nil {
		return
	}
	afterInvalidation(id)
}

func excludeChildren(node *graph.Node, excludedOptions map[string]bool) {
	for _, child := range node.Childs {
		excludedOptions[child.ID] = true
		excludeChildren(child, excludedOptions)
	}
}

func execOptionCommand(ctx context.Context, command string, resolvedOptions map[string]config.OptionValue, extraValues map[string]string) (*bytes.Buffer, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := os.Environ()
	for k, v := range combine(resolvedOptions, extraValues) {
		env = append(env, k+"="+v)
	}

	err := shell.ExecuteCommandWithShell(ctx, command, nil, stdout, stderr, env)
	if err != nil {
		return nil, errors.Wrapf(err, "exec command: %s%s", stdout.String(), stderr.String())
	}

	return stdout, nil
}

func resolveFromCommand(ctx context.Context, option *types.Option, resolvedOptions map[string]config.OptionValue, extraValues map[string]string) (config.OptionValue, error) {
	cmdOut, err := execOptionCommand(ctx, option.Command, resolvedOptions, extraValues)
	if err != nil {
		return config.OptionValue{}, errors.Wrap(err, "run command")
	}
	optionValue := config.OptionValue{Value: strings.TrimSpace(cmdOut.String())}
	expire := types.NewTime(time.Now())
	optionValue.Filled = &expire
	return optionValue, nil
}

func resolveSubOptions(ctx context.Context, option *types.Option, optionName string, resolvedOptions map[string]config.OptionValue, extraValues map[string]string) (config.OptionValue, config.DynamicOptions, error) {
	cmdOut, err := execOptionCommand(ctx, option.SubOptionsCommand, resolvedOptions, extraValues)
	if err != nil {
		return config.OptionValue{}, nil, errors.Wrap(err, "run subOptionsCommand")
	}
	subOptions := provider.SubOptions{}
	err = json.Unmarshal(cmdOut.Bytes(), &subOptions)
	if err != nil {
		return config.OptionValue{}, nil, errors.Wrapf(err, "parse subOptionsCommand: %s", cmdOut.String())
	}

	// prepare new options
	retOpts := config.DynamicOptions{}
	children := []string{}
	// need to look for option in graph. should be rather easy because we don't need to traverse the whole graph
	for k, v := range subOptions.Options {
		cp := v
		retOpts[k] = &cp
		children = append(children, k)
	}

	newOpt := resolvedOptions[optionName]
	newOpt.Children = children

	return newOpt, retOpts, nil
}

func combine(resolvedOptions map[string]config.OptionValue, extraValues map[string]string) map[string]string {
	options := map[string]string{}
	for k, v := range extraValues {
		options[k] = v
	}
	for k, v := range resolvedOptions {
		options[k] = v.Value
	}
	return options
}

func addDependencies(g *graph.Graph, options config.DynamicOptions, optionValues map[string]config.OptionValue) error {
	for optionName, option := range options {
		// Always add children as dependencies
		children := optionValues[optionName].Children
		if len(children) > 0 {
			for _, childName := range children {
				dep := options[childName]
				if dep == nil {
					continue
				}
				if option.Global && !dep.Global {
					return fmt.Errorf("cannot use a global option as a dependency of a non-global option. Option '%s' used in command of option '%s'", childName, optionName)
				} else if !option.Local && dep.Local {
					return fmt.Errorf("cannot use a non-local option as a dependency of a local option. Option '%s' used in default of option '%s'", childName, optionName)
				}
				err := g.AddEdge(optionName, childName)
				if err != nil {
					return err
				}
			}

			continue
		}

		deps := findVariables(option.Default)
		for _, dep := range deps {
			if options[dep] == nil || dep == optionName {
				continue
			}

			if option.Global && !options[dep].Global {
				return fmt.Errorf("cannot use a global option as a dependency of a non-global option. Option '%s' used in default of option '%s'", dep, optionName)
			} else if !option.Local && options[dep].Local {
				return fmt.Errorf("cannot use a non-local option as a dependency of a local option. Option '%s' used in default of option '%s'", dep, optionName)
			}

			err := g.AddEdge(dep, optionName)
			if err != nil {
				return err
			}
		}

		deps = findVariables(option.Command)
		for _, dep := range deps {
			if options[dep] == nil || dep == optionName {
				continue
			}

			if option.Global && !options[dep].Global {
				return fmt.Errorf("cannot use a global option as a dependency of a non-global option. Option '%s' used in command of option '%s'", dep, optionName)
			} else if !option.Local && options[dep].Local {
				return fmt.Errorf("cannot use a non-local option as a dependency of a local option. Option '%s' used in default of option '%s'", dep, optionName)
			}

			err := g.AddEdge(dep, optionName)
			if err != nil {
				return err
			}
		}
	}

	// remove root parent if possible
	removeRootParent(g, options)
	return nil
}

func addOptionsToGraph(g *graph.Graph, options config.DynamicOptions) error {
	for optionName, option := range options {
		_, err := g.InsertNodeAt(rootID, optionName, option)
		if err != nil {
			return err
		}
	}

	return nil
}

func removeRootParent(g *graph.Graph, options config.DynamicOptions) {
	for optionName := range options {
		node := g.Nodes[optionName]

		// remove root parent
		if len(node.Parents) > 1 {
			newParents := []*graph.Node{}
			removed := false
			for _, parent := range node.Parents {
				if parent.ID == rootID {
					removed = true
					continue
				}
				newParents = append(newParents, parent)
			}
			node.Parents = newParents

			// remove from root childs
			if removed {
				newChilds := []*graph.Node{}
				for _, child := range g.Root.Childs {
					if child.ID == node.ID {
						continue
					}
					newChilds = append(newChilds, child)
				}
				g.Root.Childs = newChilds
			}
		}
	}
}

func findVariables(str string) []string {
	retVars := map[string]bool{}
	matches := variableExpression.FindAllStringSubmatch(str, -1)
	for _, match := range matches {
		if len(match) != 5 {
			continue
		}

		retVars[match[1]] = true
	}

	retVarsArr := []string{}
	for k := range retVars {
		retVarsArr = append(retVarsArr, k)
	}

	sort.Strings(retVarsArr)
	return retVarsArr
}

func contains(stack []string, k string) bool {
	for _, s := range stack {
		if s == k {
			return true
		}
	}
	return false
}
