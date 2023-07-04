package options

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/binaries"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/graph"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/shell"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/log/terminal"
	"github.com/pkg/errors"
)

type HasOptionFunc func(name string) bool

var variableExpression = regexp.MustCompile(`(?m)\$\{?([A-Z0-9_]+)(:(-|\+)([^\}]+))?\}?`)

const (
	rootID = "root"
)

func ResolveAndSaveOptionsMachine(ctx context.Context, devConfig *config.Config, provider *provider2.ProviderConfig, originalMachine *provider2.Machine, userOptions map[string]string, log log.Logger) (*provider2.Machine, error) {
	// reload config
	machine, err := provider2.LoadMachineConfig(originalMachine.Context, originalMachine.ID)
	if err != nil {
		return originalMachine, err
	}

	// resolve devconfig options
	var beforeConfigOptions map[string]config.OptionValue
	if machine != nil {
		beforeConfigOptions = machine.Provider.Options
	}

	// get binary paths
	binaryPaths, err := binaries.GetBinaries(devConfig.DefaultContext, provider)
	if err != nil {
		return nil, err
	}

	// resolve options
	dynamicOptions := mergeProviderOptions(provider.Options, devConfig.DynamicProviderOptions(provider.Name))
	resolvedOptions, _, err := resolveOptionsGeneric(
		ctx,
		dynamicOptions,
		provider2.CombineOptions(nil, machine, devConfig.ProviderOptions(provider.Name)),
		userOptions,
		provider2.Merge(provider2.ToOptionsMachine(machine), binaryPaths),
		true,
		false,
		false,
		log,
	)
	if err != nil {
		return nil, err
	}

	// remove global options
	filterResolvedOptions(resolvedOptions, beforeConfigOptions, devConfig.ProviderOptions(provider.Name), provider.Options, userOptions)

	// save machine config
	if machine != nil {
		machine.Provider.Options = resolvedOptions

		if !reflect.DeepEqual(beforeConfigOptions, machine.Provider.Options) {
			err = provider2.SaveMachineConfig(machine)
			if err != nil {
				return machine, err
			}
		}
	}

	return machine, nil
}

func ResolveAndSaveOptionsWorkspace(ctx context.Context, devConfig *config.Config, provider *provider2.ProviderConfig, originalWorkspace *provider2.Workspace, userOptions map[string]string, log log.Logger) (*provider2.Workspace, error) {
	// reload config
	workspace, err := provider2.LoadWorkspaceConfig(originalWorkspace.Context, originalWorkspace.ID)
	if err != nil {
		return originalWorkspace, err
	}

	// resolve devconfig options
	var beforeConfigOptions map[string]config.OptionValue
	if workspace != nil {
		beforeConfigOptions = workspace.Provider.Options
	}

	// get binary paths
	binaryPaths, err := binaries.GetBinaries(devConfig.DefaultContext, provider)
	if err != nil {
		return nil, err
	}

	// resolve options
	dynamicOptions := mergeProviderOptions(provider.Options, devConfig.DynamicProviderOptions(provider.Name))
	resolvedOptions, _, err := resolveOptionsGeneric(
		ctx,
		dynamicOptions,
		provider2.CombineOptions(workspace, nil, devConfig.ProviderOptions(provider.Name)),
		userOptions,
		provider2.Merge(provider2.ToOptionsWorkspace(workspace), binaryPaths),
		true,
		false,
		false,
		log,
	)
	if err != nil {
		return nil, err
	}

	// remove global options
	filterResolvedOptions(resolvedOptions, beforeConfigOptions, devConfig.ProviderOptions(provider.Name), provider.Options, userOptions)

	// save workspace config
	if workspace != nil {
		workspace.Provider.Options = resolvedOptions

		if !reflect.DeepEqual(beforeConfigOptions, workspace.Provider.Options) {
			err = provider2.SaveWorkspaceConfig(workspace)
			if err != nil {
				return workspace, err
			}
		}
	}

	return workspace, nil
}

func ResolveOptions(ctx context.Context, devConfig *config.Config, provider *provider2.ProviderConfig, userOptions map[string]string, skipRequired bool, singleMachine *bool, init bool, log log.Logger) (*config.Config, error) {
	// get binary paths
	binaryPaths, err := binaries.GetBinaries(devConfig.DefaultContext, provider)
	if err != nil {
		return nil, err
	}

	options := mergeProviderOptions(provider.Options, devConfig.DynamicProviderOptions(provider.Name))
	dynamicOptions := config.DynamicOptions{}
	resolvedOptions := devConfig.ProviderOptions(provider.Name)

	isDone := func() bool {
		// check if dynamic options are all resolved
		for k, v := range dynamicOptions {
			resOpt, ok := resolvedOptions[k]

			if v.Required && (!ok || resOpt.Value == "") {
				return false
			}
		}

		return true
	}

	for stop := false; !stop; stop = isDone() {
		newResOpts, newDynOpts, err := resolveOptionsGeneric(
			ctx,
			options,
			resolvedOptions,
			userOptions,
			provider2.Merge(provider2.GetBaseEnvironment(devConfig.DefaultContext, provider.Name), binaryPaths),
			false,
			true,
			skipRequired,
			log,
		)
		if err != nil {
			return nil, err
		}
		dynamicOptions = mergeOptions(dynamicOptions, newDynOpts)
		resolvedOptions = newResOpts

		if !init {
			break
		}
		// prepare next tick
		options = mergeOptions(options, dynamicOptions)
	}

	// dev config
	if devConfig != nil {
		devConfig = config.CloneConfig(devConfig)
		if devConfig.Current().Providers == nil {
			devConfig.Current().Providers = map[string]*config.ProviderConfig{}
		}
		if devConfig.Current().Providers[provider.Name] == nil {
			devConfig.Current().Providers[provider.Name] = &config.ProviderConfig{}
		}
		devConfig.Current().Providers[provider.Name].Options = map[string]config.OptionValue{}
		for k, v := range resolvedOptions {
			devConfig.Current().Providers[provider.Name].Options[k] = v
		}

		devConfig.Current().Providers[provider.Name].DynamicOptions = config.DynamicOptions{}
		for k, v := range dynamicOptions {
			devConfig.Current().Providers[provider.Name].DynamicOptions[k] = v
		}
		if singleMachine != nil {
			devConfig.Current().Providers[provider.Name].SingleMachine = *singleMachine
		}
	}

	return devConfig, nil
}

func ResolveAgentConfig(devConfig *config.Config, provider *provider2.ProviderConfig, workspace *provider2.Workspace, machine *provider2.Machine) provider2.ProviderAgentConfig {
	// fill in agent config
	options := provider2.ToOptions(workspace, machine, devConfig.ProviderOptions(provider.Name))
	agentConfig := provider.Agent
	agentConfig.Driver = resolveDefaultValue(agentConfig.Driver, options)
	agentConfig.Local = types.StrBool(resolveDefaultValue(string(agentConfig.Local), options))
	agentConfig.Docker.Path = resolveDefaultValue(agentConfig.Docker.Path, options)
	agentConfig.Docker.Install = types.StrBool(resolveDefaultValue(string(agentConfig.Docker.Install), options))
	agentConfig.Docker.Env = resolveDefaultValues(agentConfig.Docker.Env, options)
	agentConfig.Kubernetes.Path = resolveDefaultValue(agentConfig.Kubernetes.Path, options)
	agentConfig.Kubernetes.HelperImage = resolveDefaultValue(agentConfig.Kubernetes.HelperImage, options)
	agentConfig.Kubernetes.Config = resolveDefaultValue(agentConfig.Kubernetes.Config, options)
	agentConfig.Kubernetes.Context = resolveDefaultValue(agentConfig.Kubernetes.Context, options)
	agentConfig.Kubernetes.Namespace = resolveDefaultValue(agentConfig.Kubernetes.Namespace, options)
	agentConfig.Kubernetes.ClusterRole = resolveDefaultValue(agentConfig.Kubernetes.ClusterRole, options)
	agentConfig.Kubernetes.ServiceAccount = resolveDefaultValue(agentConfig.Kubernetes.ServiceAccount, options)
	agentConfig.Kubernetes.BuildRepository = resolveDefaultValue(agentConfig.Kubernetes.BuildRepository, options)
	agentConfig.Kubernetes.BuildkitImage = resolveDefaultValue(agentConfig.Kubernetes.BuildkitImage, options)
	agentConfig.Kubernetes.BuildkitPrivileged = types.StrBool(resolveDefaultValue(string(agentConfig.Kubernetes.BuildkitPrivileged), options))
	agentConfig.Kubernetes.PersistentVolumeSize = resolveDefaultValue(agentConfig.Kubernetes.PersistentVolumeSize, options)
	agentConfig.Kubernetes.StorageClassName = resolveDefaultValue(agentConfig.Kubernetes.StorageClassName, options)
	agentConfig.Kubernetes.NodeSelector = resolveDefaultValue(agentConfig.Kubernetes.NodeSelector, options)
	agentConfig.Kubernetes.BuildkitNodeSelector = resolveDefaultValue(agentConfig.Kubernetes.BuildkitNodeSelector, options)
	agentConfig.Kubernetes.Resources = resolveDefaultValue(agentConfig.Kubernetes.Resources, options)
	agentConfig.Kubernetes.HelperResources = resolveDefaultValue(agentConfig.Kubernetes.HelperResources, options)
	agentConfig.Kubernetes.BuildkitResources = resolveDefaultValue(agentConfig.Kubernetes.BuildkitResources, options)
	agentConfig.Kubernetes.CreateNamespace = types.StrBool(resolveDefaultValue(string(agentConfig.Kubernetes.CreateNamespace), options))
	agentConfig.DataPath = resolveDefaultValue(agentConfig.DataPath, options)
	agentConfig.Path = resolveDefaultValue(agentConfig.Path, options)
	if agentConfig.Path == "" && agentConfig.Local == "true" {
		agentConfig.Path, _ = os.Executable()
	} else if agentConfig.Path == "" {
		agentConfig.Path = agent.RemoteDevPodHelperLocation
	}
	agentConfig.DownloadURL = resolveDefaultValue(agentConfig.DownloadURL, options)
	if agentConfig.DownloadURL == "" {
		agentConfig.DownloadURL = agent.DefaultAgentDownloadURL()
	}
	agentConfig.Timeout = resolveDefaultValue(agentConfig.Timeout, options)
	agentConfig.ContainerTimeout = resolveDefaultValue(agentConfig.ContainerTimeout, options)
	agentConfig.InjectGitCredentials = types.StrBool(resolveDefaultValue(string(agentConfig.InjectGitCredentials), options))
	agentConfig.InjectDockerCredentials = types.StrBool(resolveDefaultValue(string(agentConfig.InjectDockerCredentials), options))
	return agentConfig
}

func mergeProviderOptions(existing map[string]*provider2.ProviderOption, newOpts config.DynamicOptions) config.DynamicOptions {
	retOptions := config.DynamicOptions{}
	for k, v := range existing {
		retOptions[k] = &v.Option
	}
	for k, v := range newOpts {
		retOptions[k] = v
	}

	return retOptions
}

func mergeOptions[K comparable, V any](existing map[K]V, newOpts map[K]V) map[K]V {
	retOpts := map[K]V{}
	for k, v := range existing {
		retOpts[k] = v
	}
	for k, v := range newOpts {
		retOpts[k] = v
	}

	return retOpts
}

func resolveOptionsGeneric(
	ctx context.Context,
	options config.DynamicOptions,
	optionValues map[string]config.OptionValue,
	userOptions map[string]string,
	extraValues map[string]string,
	resolveLocal bool,
	resolveGlobal bool,
	skipRequired bool,
	log log.Logger,
) (map[string]config.OptionValue, config.DynamicOptions, error) {
	if options == nil {
		options = config.DynamicOptions{}
	}
	if userOptions == nil {
		userOptions = map[string]string{}
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

	// resolve options
	resolvedOptions, dynamicOptions, err := resolveOptions(
		ctx,
		g,
		optionValues,
		userOptions,
		extraValues,
		resolveLocal,
		resolveGlobal,
		skipRequired,
		log,
	)
	if err != nil {
		return nil, nil, err
	}

	return resolvedOptions, dynamicOptions, nil
}

func resolveOptions(
	ctx context.Context,
	g *graph.Graph,
	optionValues map[string]config.OptionValue,
	userOptions map[string]string,
	extraValues map[string]string,
	resolveLocal bool,
	resolveGlobal bool,
	skipRequired bool,
	log log.Logger,
) (map[string]config.OptionValue, config.DynamicOptions, error) {
	// copy options
	resolvedOptions := map[string]config.OptionValue{}
	for optionName, v := range optionValues {
		resolvedOptions[optionName] = v
	}

	// resolve options order
	clonedGraph := g.Clone()
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

		newOpts, err := resolveOption(ctx, g, optionName, resolvedOptions, excludedOptions, userOptions, extraValues, resolveLocal, resolveGlobal, skipRequired, log)
		if err != nil {
			return nil, nil, errors.Wrap(err, "resolve option "+optionName)
		}
		for k, v := range newOpts {
			dynamicOptions[k] = v
		}
	}

	return resolvedOptions, dynamicOptions, nil
}

func resolveOption(
	ctx context.Context,
	g *graph.Graph,
	optionName string,
	resolvedOptions map[string]config.OptionValue,
	excludedOptions map[string]bool,
	userOptions map[string]string,
	extraValues map[string]string,
	resolveLocal bool,
	resolveGlobal bool,
	skipRequired bool,
	log log.Logger,
) (config.DynamicOptions, error) {
	dynamicOptions := config.DynamicOptions{}
	node := g.Nodes[optionName]
	option := node.Data.(*types.Option)

	// check if user value exists
	userValue, userValueOk := userOptions[optionName]

	// find out options we need to resolve
	if !userValueOk {
		// make sure required is always resolved
		if !option.Required {
			// skip if global
			if !resolveGlobal && option.Global {
				return dynamicOptions, nil
			} else if !resolveLocal && option.Local {
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
			Value: resolveDefaultValue(option.Default, combine(resolvedOptions, extraValues)),
		}
	} else if option.Command != "" {
		optionValue, err := resolveFromCommand(ctx, option, resolvedOptions, extraValues)
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
		if skipRequired {
			delete(resolvedOptions, optionName)
			excludeChildren(g.Nodes[optionName], excludedOptions)
			return dynamicOptions, nil
		}

		// check if we can ask a question
		if !terminal.IsTerminalIn {
			return dynamicOptions, fmt.Errorf("option %s is required, but no value provided", optionName)
		}

		log.Info(option.Description)
		answer, err := log.Question(&survey.QuestionOptions{
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
			updatedOpt, newOpts, err := resolveSubOptions(ctx, option, optionName, resolvedOptions, extraValues)
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
				invalidateSubGraph(g, childID, func(id string) {
					delete(resolvedOptions, id)
					delete(userOptions, id)
					excludedOptions[id] = true
				})
			}
		} else {
			excludeChildren(g.Nodes[optionName], excludedOptions)
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

func resolveFromCommand(ctx context.Context, option *types.Option, resolvedOptions map[string]config.OptionValue, extraValues map[string]string) (config.OptionValue, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := os.Environ()
	for k, v := range combine(resolvedOptions, extraValues) {
		env = append(env, k+"="+v)
	}

	err := shell.ExecuteCommandWithShell(ctx, option.Command, nil, stdout, stderr, env)
	if err != nil {
		return config.OptionValue{}, errors.Wrapf(err, "run command: %s%s", stdout.String(), stderr.String())
	}

	optionValue := config.OptionValue{Value: strings.TrimSpace(stdout.String())}
	expire := types.NewTime(time.Now())
	optionValue.Filled = &expire
	return optionValue, nil
}

func resolveSubOptions(ctx context.Context, option *types.Option, optionName string, resolvedOptions map[string]config.OptionValue, extraValues map[string]string) (config.OptionValue, config.DynamicOptions, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := os.Environ()
	for k, v := range combine(resolvedOptions, extraValues) {
		env = append(env, k+"="+v)
	}

	err := shell.ExecuteCommandWithShell(ctx, option.SubOptionsCommand, nil, stdout, stderr, env)
	if err != nil {
		return config.OptionValue{}, nil, errors.Wrapf(err, "run subOptionsCommand: %s%s", stdout.String(), stderr.String())
	}
	subOptions := provider2.SubOptions{}
	err = json.Unmarshal(stdout.Bytes(), &subOptions)
	if err != nil {
		return config.OptionValue{}, nil, errors.Wrapf(err, "parse subOptionsCommand: %s", stdout.String())
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

// replace all value in the map with the resolved default value
func resolveDefaultValues(vals map[string]string, resolvedOptions map[string]string) map[string]string {
	ret := make(map[string]string)
	for k, v := range vals {
		resolvedValue := resolveDefaultValue(v, resolvedOptions)
		if resolvedValue == "" {
			continue
		}

		ret[k] = resolvedValue
	}
	return ret
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

		deps := FindVariables(option.Default)
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

		deps = FindVariables(option.Command)
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

func filterResolvedOptions(resolvedOptions, beforeConfigOptions, providerValues map[string]config.OptionValue, providerOptions map[string]*provider2.ProviderOption, userOptions map[string]string) {
	for k := range resolvedOptions {
		// check if user supplied
		if userOptions != nil {
			_, ok := userOptions[k]
			if ok {
				continue
			}
		}

		// check if it was there before
		if beforeConfigOptions != nil {
			_, ok := beforeConfigOptions[k]
			if ok {
				continue
			}
		}

		// check if not available in the provider values
		if providerValues != nil {
			_, ok := providerValues[k]
			if !ok {
				continue
			}
		}

		// check if not global
		if providerOptions == nil || providerOptions[k] == nil || !providerOptions[k].Global {
			continue
		}

		delete(resolvedOptions, k)
	}
}

func contains(stack []string, k string) bool {
	for _, s := range stack {
		if s == k {
			return true
		}
	}
	return false
}
