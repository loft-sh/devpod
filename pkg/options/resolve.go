package options

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/graph"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/shell"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

var variableExpression = regexp.MustCompile(`(?m)\$\{?([A-Z0-9_]+)(:(-|\+)([^\}]+))?\}?`)

const rootID = "root"

func ResolveAndSaveOptions(ctx context.Context, beforeStage, afterStage string, originalDevConfig *config.Config, provider *provider2.ProviderConfig) (*config.Config, error) {
	// reload config
	devConfig, err := config.LoadConfig(originalDevConfig.DefaultContext)
	if err != nil {
		return originalDevConfig, err
	}

	// resolve devconfig options
	var beforeConfigOptions map[string]config.OptionValue
	if devConfig != nil {
		beforeConfigOptions = devConfig.Current().ProviderOptions(provider.Name)
	}

	// resolve options
	devConfig, err = ResolveOptions(ctx, beforeStage, afterStage, devConfig, provider)
	if err != nil {
		return devConfig, errors.Wrap(err, "resolve options")
	}

	// save devconfig config
	if devConfig != nil && !reflect.DeepEqual(devConfig.Current().ProviderOptions(provider.Name), beforeConfigOptions) {
		err = config.SaveConfig(devConfig)
		if err != nil {
			return devConfig, err
		}
	}

	return devConfig, nil
}

func ResolveOptions(ctx context.Context, beforeStage, afterStage string, devConfig *config.Config, provider *provider2.ProviderConfig) (*config.Config, error) {
	resolvedOptions, err := resolveOptionsGeneric(ctx, beforeStage, afterStage, devConfig.ProviderOptions(provider.Name), extraOptions(), provider)
	if err != nil {
		return nil, err
	}

	// dev config
	if devConfig != nil {
		devConfig = config.CloneConfig(devConfig)
		if devConfig.Current().Providers[provider.Name] == nil {
			devConfig.Current().Providers[provider.Name] = &config.ConfigProvider{}
		}
		devConfig.Current().Providers[provider.Name].Options = map[string]config.OptionValue{}
		for k, v := range resolvedOptions {
			devConfig.Current().Providers[provider.Name].Options[k] = v
		}
	}

	return devConfig, nil
}

func extraOptions() map[string]string {
	retVars := map[string]string{}
	devPodBinary, _ := os.Executable()
	retVars[provider2.DEVPOD] = filepath.ToSlash(devPodBinary)
	return retVars
}

func resolveOptionsGeneric(ctx context.Context, beforeStage, afterStage string, optionValues map[string]config.OptionValue, extraValues map[string]string, provider *provider2.ProviderConfig) (map[string]config.OptionValue, error) {
	options := provider.Options
	if options == nil {
		options = map[string]*provider2.ProviderOption{}
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
	resolvedOptions, err := resolveOptions(ctx, g, beforeStage, afterStage, options, optionValues, extraValues)
	if err != nil {
		return nil, err
	}

	return resolvedOptions, nil
}

func ResolveAgentConfig(devConfig *config.Config, provider *provider2.ProviderConfig) provider2.ProviderAgentConfig {
	// fill in agent config
	options := provider2.ToOptions(nil, nil, devConfig.ProviderOptions(provider.Name))
	agentConfig := provider.Agent
	agentConfig.Path = resolveDefaultValue(agentConfig.Path, options)
	if agentConfig.Path == "" {
		agentConfig.Path = agent.RemoteDevPodHelperLocation
	}
	agentConfig.DownloadURL = resolveDefaultValue(agentConfig.DownloadURL, options)
	if agentConfig.DownloadURL == "" {
		agentConfig.DownloadURL = agent.DefaultAgentDownloadURL
	}
	agentConfig.Timeout = resolveDefaultValue(agentConfig.Timeout, options)
	agentConfig.InjectGitCredentials = types.StrBool(resolveDefaultValue(string(agentConfig.InjectGitCredentials), options))
	agentConfig.InjectDockerCredentials = types.StrBool(resolveDefaultValue(string(agentConfig.InjectDockerCredentials), options))
	return agentConfig
}

func resolveOptions(ctx context.Context, g *graph.Graph, beforeStage, afterStage string, options map[string]*provider2.ProviderOption, optionValues map[string]config.OptionValue, extraValues map[string]string) (map[string]config.OptionValue, error) {
	// find out options we need to resolve
	resolveOptions := map[string]bool{}
	for optionName, option := range options {
		if option.Before != beforeStage || option.After != afterStage {
			continue
		}

		if optionValues != nil {
			val, ok := optionValues[optionName]
			if ok && (val.Expires == nil || time.Now().Before(val.Expires.Time)) {
				continue
			}
		}

		resolveOptions[optionName] = true
	}

	// resolve options
	resolvedOptions := map[string]config.OptionValue{}
	if optionValues != nil {
		for optionName, v := range optionValues {
			if resolveOptions[optionName] {
				continue
			}

			resolvedOptions[optionName] = v
		}
	}

	// resolve options
	for optionName := range resolveOptions {
		err := resolveOption(ctx, g, optionName, resolveOptions, resolvedOptions, extraValues)
		if err != nil {
			return nil, errors.Wrap(err, "resolve option "+optionName)
		}
	}

	// TODO: recompute children?
	return resolvedOptions, nil
}

func resolveOption(ctx context.Context, g *graph.Graph, optionName string, resolveOptions map[string]bool, resolvedOptions map[string]config.OptionValue, extraValues map[string]string) error {
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
		err := resolveOption(ctx, g, parent.ID, resolveOptions, resolvedOptions, extraValues)
		if err != nil {
			return err
		}
	}

	// resolve option
	option := node.Data.(*provider2.ProviderOption)
	if option.Default != "" {
		resolvedOptions[optionName] = config.OptionValue{
			Value: resolveDefaultValue(option.Default, combine(resolvedOptions, extraValues)),
		}
	} else if option.Command != "" {
		optionValue, err := resolveFromCommand(ctx, option, resolvedOptions, extraValues)
		if err != nil {
			return err
		}

		resolvedOptions[optionName] = optionValue
	} else {
		resolvedOptions[optionName] = config.OptionValue{}
	}

	return nil
}

func resolveFromCommand(ctx context.Context, option *provider2.ProviderOption, resolvedOptions map[string]config.OptionValue, extraValues map[string]string) (config.OptionValue, error) {
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
	if option.Cache != "" {
		duration, err := time.ParseDuration(option.Cache)
		if err != nil {
			return config.OptionValue{}, errors.Wrap(err, "parse cache duration")
		}

		expire := types.NewTime(time.Now().Add(duration))
		optionValue.Expires = &expire
	} else {
		expire := types.Now()
		optionValue.Expires = &expire
	}

	return optionValue, nil
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
