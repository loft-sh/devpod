package options

import (
	"context"
	"os"
	"reflect"
	"strings"

	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/binaries"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/options/resolver"

	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/log"
)

func ResolveAndSaveOptionsMachine(
	ctx context.Context,
	devConfig *config.Config,
	provider *provider2.ProviderConfig,
	originalMachine *provider2.Machine,
	userOptions map[string]string,
	log log.Logger,
) (*provider2.Machine, error) {
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
	resolvedOptions, _, err := resolver.New(
		userOptions,
		provider2.Merge(provider2.ToOptionsMachine(machine), binaryPaths),
		log,
		resolver.WithResolveLocal(),
	).Resolve(
		ctx,
		devConfig.DynamicProviderOptionDefinitions(provider.Name),
		provider.Options,
		provider2.CombineOptions(nil, machine, devConfig.ProviderOptions(provider.Name)),
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

func ResolveAndSaveOptionsWorkspace(
	ctx context.Context,
	devConfig *config.Config,
	provider *provider2.ProviderConfig,
	originalWorkspace *provider2.Workspace,
	userOptions map[string]string,
	log log.Logger,
) (*provider2.Workspace, error) {
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
	resolvedOptions, _, err := resolver.New(
		userOptions,
		provider2.Merge(provider2.ToOptionsWorkspace(workspace), binaryPaths),
		log,
		resolver.WithResolveLocal(),
	).Resolve(
		ctx,
		devConfig.DynamicProviderOptionDefinitions(provider.Name),
		provider.Options,
		provider2.CombineOptions(workspace, nil, devConfig.ProviderOptions(provider.Name)),
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

func ResolveOptions(
	ctx context.Context,
	devConfig *config.Config,
	provider *provider2.ProviderConfig,
	userOptions map[string]string,
	skipRequired bool,
	singleMachine *bool,
	log log.Logger,
) (*config.Config, error) {
	// get binary paths
	binaryPaths, err := binaries.GetBinaries(devConfig.DefaultContext, provider)
	if err != nil {
		return nil, err
	}

	// create new resolver
	resolve := resolver.New(
		userOptions,
		provider2.Merge(provider2.GetBaseEnvironment(devConfig.DefaultContext, provider.Name), binaryPaths),
		log,
		resolver.WithResolveGlobal(),
		resolver.WithResolveSubOptions(),
		resolver.WithSkipRequired(skipRequired),
	)

	// loop and resolve options, as soon as we encounter a new dynamic option it will get filled
	resolvedOptionValues, dynamicOptionDefinitions, err := resolve.Resolve(
		ctx,
		devConfig.DynamicProviderOptionDefinitions(provider.Name),
		provider.Options,
		devConfig.ProviderOptions(provider.Name),
	)
	if err != nil {
		return nil, err
	}

	// save options in dev config
	if devConfig != nil {
		devConfig = config.CloneConfig(devConfig)
		if devConfig.Current().Providers == nil {
			devConfig.Current().Providers = map[string]*config.ProviderConfig{}
		}
		if devConfig.Current().Providers[provider.Name] == nil {
			devConfig.Current().Providers[provider.Name] = &config.ProviderConfig{}
		}
		devConfig.Current().Providers[provider.Name].Options = map[string]config.OptionValue{}
		for k, v := range resolvedOptionValues {
			devConfig.Current().Providers[provider.Name].Options[k] = v
		}

		devConfig.Current().Providers[provider.Name].DynamicOptions = config.OptionDefinitions{}
		for k, v := range dynamicOptionDefinitions {
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
	agentConfig.Driver = resolver.ResolveDefaultValue(agentConfig.Driver, options)
	agentConfig.Local = types.StrBool(resolver.ResolveDefaultValue(string(agentConfig.Local), options))
	agentConfig.Docker.Path = resolver.ResolveDefaultValue(agentConfig.Docker.Path, options)
	agentConfig.Docker.Install = types.StrBool(resolver.ResolveDefaultValue(string(agentConfig.Docker.Install), options))
	agentConfig.Docker.Env = resolver.ResolveDefaultValues(agentConfig.Docker.Env, options)
	agentConfig.Kubernetes.Path = resolver.ResolveDefaultValue(agentConfig.Kubernetes.Path, options)
	agentConfig.Kubernetes.HelperImage = resolver.ResolveDefaultValue(agentConfig.Kubernetes.HelperImage, options)
	agentConfig.Kubernetes.Config = resolver.ResolveDefaultValue(agentConfig.Kubernetes.Config, options)
	agentConfig.Kubernetes.Context = resolver.ResolveDefaultValue(agentConfig.Kubernetes.Context, options)
	agentConfig.Kubernetes.Namespace = resolver.ResolveDefaultValue(agentConfig.Kubernetes.Namespace, options)
	agentConfig.Kubernetes.ClusterRole = resolver.ResolveDefaultValue(agentConfig.Kubernetes.ClusterRole, options)
	agentConfig.Kubernetes.ServiceAccount = resolver.ResolveDefaultValue(agentConfig.Kubernetes.ServiceAccount, options)
	agentConfig.Kubernetes.BuildRepository = resolver.ResolveDefaultValue(agentConfig.Kubernetes.BuildRepository, options)
	agentConfig.Kubernetes.BuildkitImage = resolver.ResolveDefaultValue(agentConfig.Kubernetes.BuildkitImage, options)
	agentConfig.Kubernetes.BuildkitPrivileged = types.StrBool(resolver.ResolveDefaultValue(string(agentConfig.Kubernetes.BuildkitPrivileged), options))
	agentConfig.Kubernetes.PersistentVolumeSize = resolver.ResolveDefaultValue(agentConfig.Kubernetes.PersistentVolumeSize, options)
	agentConfig.Kubernetes.PVCAccessMode = resolver.ResolveDefaultValue(agentConfig.Kubernetes.PVCAccessMode, options)
	agentConfig.Kubernetes.StorageClassName = resolver.ResolveDefaultValue(agentConfig.Kubernetes.StorageClassName, options)
	agentConfig.Kubernetes.PodManifestTemplate = resolver.ResolveDefaultValue(agentConfig.Kubernetes.PodManifestTemplate, options)
	agentConfig.Kubernetes.NodeSelector = resolver.ResolveDefaultValue(agentConfig.Kubernetes.NodeSelector, options)
	agentConfig.Kubernetes.BuildkitNodeSelector = resolver.ResolveDefaultValue(agentConfig.Kubernetes.BuildkitNodeSelector, options)
	agentConfig.Kubernetes.Resources = resolver.ResolveDefaultValue(agentConfig.Kubernetes.Resources, options)
	agentConfig.Kubernetes.Labels = resolver.ResolveDefaultValue(agentConfig.Kubernetes.Labels, options)
	agentConfig.Kubernetes.HelperResources = resolver.ResolveDefaultValue(agentConfig.Kubernetes.HelperResources, options)
	agentConfig.Kubernetes.BuildkitResources = resolver.ResolveDefaultValue(agentConfig.Kubernetes.BuildkitResources, options)
	agentConfig.Kubernetes.CreateNamespace = types.StrBool(resolver.ResolveDefaultValue(string(agentConfig.Kubernetes.CreateNamespace), options))
	agentConfig.DataPath = resolver.ResolveDefaultValue(agentConfig.DataPath, options)
	agentConfig.Path = resolver.ResolveDefaultValue(agentConfig.Path, options)
	if agentConfig.Path == "" && agentConfig.Local == "true" {
		agentConfig.Path, _ = os.Executable()
	} else if agentConfig.Path == "" {
		agentConfig.Path = agent.RemoteDevPodHelperLocation
	}
	agentConfig.DownloadURL = resolver.ResolveDefaultValue(agentConfig.DownloadURL, options)
	if agentConfig.DownloadURL == "" {
		agentConfig.DownloadURL = resolveAgentDownloadURL(devConfig)
	}
	agentConfig.Timeout = resolver.ResolveDefaultValue(agentConfig.Timeout, options)
	agentConfig.ContainerTimeout = resolver.ResolveDefaultValue(agentConfig.ContainerTimeout, options)
	agentConfig.InjectGitCredentials = types.StrBool(resolver.ResolveDefaultValue(string(agentConfig.InjectGitCredentials), options))
	agentConfig.InjectDockerCredentials = types.StrBool(resolver.ResolveDefaultValue(string(agentConfig.InjectDockerCredentials), options))
	return agentConfig
}

// resolveAgentDownloadURL resolves the agent download URL (env -> context -> default)
func resolveAgentDownloadURL(devConfig *config.Config) string {
	devPodAgentURL := os.Getenv(agent.EnvDevPodAgentURL)
	if devPodAgentURL != "" {
		return strings.TrimSuffix(devPodAgentURL, "/") + "/"
	}

	contextAgentOption, ok := devConfig.Current().Options[config.ContextOptionAgentURL]
	if ok && contextAgentOption.Value != "" {
		return strings.TrimSuffix(contextAgentOption.Value, "/") + "/"
	}

	return agent.DefaultAgentDownloadURL()
}

func filterResolvedOptions(resolvedOptions, beforeConfigOptions, providerValues map[string]config.OptionValue, providerOptions map[string]*types.Option, userOptions map[string]string) {
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
