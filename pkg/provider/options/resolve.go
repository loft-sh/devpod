package options

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/devcontainer/graph"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/shell"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/pkg/errors"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

var variableExpression = regexp.MustCompile(`(?m)\$\{?([A-Z0-9_]+)(:(-|\+)([^\}]+))?\}?`)

const rootID = "root"

func ResolveOptions(ctx context.Context, beforeStage, afterStage string, workspace *provider2.Workspace, options map[string]*provider2.ProviderOption) (map[string]provider2.OptionValue, error) {
	if options == nil {
		options = map[string]*provider2.ProviderOption{}
	}
	if workspace != nil && workspace.Provider.Options == nil {
		workspace.Provider.Options = map[string]provider2.OptionValue{}
	}

	// create a new graph
	g := graph.NewGraphOf(graph.NewNode(rootID, nil), "provider option")

	// first add all options to the graph
	err := addOptionsToGraph(g, options)
	if err != nil {
		return nil, err
	}

	// next add the dependencies
	err = addDependencies(g, options)
	if err != nil {
		return nil, err
	}

	// resolve options
	resolvedOptions, err := resolveOptions(ctx, g, beforeStage, afterStage, options, workspace)
	if err != nil {
		return nil, err
	}

	return resolvedOptions, nil
}

func resolveOptions(ctx context.Context, g *graph.Graph, beforeStage, afterStage string, options map[string]*provider2.ProviderOption, workspace *provider2.Workspace) (map[string]provider2.OptionValue, error) {
	// find out options we need to resolve
	resolveOptions := map[string]bool{}
	for optionName, option := range options {
		if option.Before != beforeStage || option.After != afterStage {
			continue
		}

		if workspace != nil {
			val, ok := workspace.Provider.Options[optionName]
			if ok && (val.Expires == nil || time.Now().Before(val.Expires.Time)) {
				continue
			}
		}

		resolveOptions[optionName] = true
	}

	// resolve options
	resolvedOptions := map[string]provider2.OptionValue{}
	if workspace != nil {
		for optionName, v := range workspace.Provider.Options {
			if resolveOptions[optionName] {
				continue
			}

			resolvedOptions[optionName] = v
		}
	}

	// resolve options
	for optionName := range resolveOptions {
		err := resolveOption(ctx, g, optionName, resolveOptions, resolvedOptions, workspace)
		if err != nil {
			return nil, errors.Wrap(err, "resolve option "+optionName)
		}
	}

	// TODO: recompute children?
	return resolvedOptions, nil
}

func resolveOption(ctx context.Context, g *graph.Graph, optionName string, resolveOptions map[string]bool, resolvedOptions map[string]provider2.OptionValue, workspace *provider2.Workspace) error {
	node := g.Nodes[optionName]

	// are parents resolved?
	for _, parent := range node.Parents {
		if parent.ID == rootID {
			continue
		}

		_, ok := resolveOptions[parent.ID]
		if !ok {
			// check if it was already resolved
			_, ok := resolvedOptions[parent.ID]
			if !ok {
				return fmt.Errorf("cannot resolve option %s, because it depends on %s which is not loaded at this stage", optionName, parent.ID)
			}

			continue
		}

		// resolve parent first
		err := resolveOption(ctx, g, parent.ID, resolveOptions, resolvedOptions, workspace)
		if err != nil {
			return err
		}
	}

	// resolve option
	option := node.Data.(*provider2.ProviderOption)
	if option.Default != "" {
		resolvedOptions[optionName] = provider2.OptionValue{
			Value: resolveDefaultValue(option.Default, toOptions(resolvedOptions, workspace)),
		}
	} else if option.Command != "" {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		env := os.Environ()
		for k, v := range toOptions(resolvedOptions, workspace) {
			env = append(env, k+"="+v)
		}

		err := shell.ExecuteCommandWithShell(ctx, option.Command, nil, stdout, stderr, env)
		if err != nil {
			return errors.Wrapf(err, "run command: %s%s", stdout.String(), stderr.String())
		}

		optionValue := provider2.OptionValue{Value: strings.TrimSpace(stdout.String())}
		if option.Cache != "" {
			duration, err := time.ParseDuration(option.Cache)
			if err != nil {
				return errors.Wrap(err, "parse cache duration")
			}

			expire := types.NewTime(time.Now().Add(duration))
			optionValue.Expires = &expire
		} else {
			expire := types.Now()
			optionValue.Expires = &expire
		}

		resolvedOptions[optionName] = optionValue
	} else {
		resolvedOptions[optionName] = provider2.OptionValue{}
	}

	return nil
}

func toOptions(resolvedOptions map[string]provider2.OptionValue, workspace *provider2.Workspace) map[string]string {
	options := map[string]string{}
	for k, v := range resolvedOptions {
		options[k] = v.Value
	}
	if workspace != nil {
		for k, v := range provider2.ToOptions(workspace) {
			options[k] = v
		}
	}
	return options
}

func resolveDefaultValue(val string, resolvedOptions map[string]string) string {
	return variableExpression.ReplaceAllStringFunc(val, func(s string) string {
		submatch := variableExpression.FindStringSubmatch(s)
		optionVal, ok := resolvedOptions[submatch[1]]
		if ok {
			return optionVal
		}

		return s
	})
}

func addDependencies(g *graph.Graph, options map[string]*provider2.ProviderOption) error {
	for optionName, option := range options {
		deps := FindVariables(option.Default)
		for _, dep := range deps {
			if options[dep] == nil || dep == optionName {
				continue
			}

			err := g.AddEdge(dep, optionName)
			if err != nil {
				return err
			}
		}

		deps = FindVariables(option.Command)
		for _, dep := range deps {
			if options[dep] == nil || dep == optionName {
				continue
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

func addOptionsToGraph(g *graph.Graph, options map[string]*provider2.ProviderOption) error {
	for optionName, option := range options {
		_, err := g.InsertNodeAt(rootID, optionName, option)
		if err != nil {
			return err
		}
	}

	return nil
}

func removeRootParent(g *graph.Graph, options map[string]*provider2.ProviderOption) {
	for optionName := range options {
		node := g.Nodes[optionName]

		// remove root parent
		if len(node.Parents) > 1 {
			newParents := []*graph.Node{}
			for _, parent := range node.Parents {
				if parent.ID == rootID {
					continue
				}
				newParents = append(newParents, parent)
			}
			node.Parents = newParents
		}
	}
}

func FindVariables(str string) []string {
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
