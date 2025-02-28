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
	options ...resolver.Option,
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
	options = append(options, resolver.WithResolveLocal())

	// resolve options
	resolvedOptions, _, err := resolver.New(
		userOptions,
		provider2.Merge(provider2.ToOptionsWorkspace(workspace), binaryPaths),
		log,
		options...,
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

func ResolveAndSaveOptionsProxy(
	ctx context.Context,
	devConfig *config.Config,
	provider *provider2.ProviderConfig,
	originalWorkspace *provider2.Workspace,
	userOptions map[string]string,
	log log.Logger,
) (*provider2.Workspace, error) {
	return ResolveAndSaveOptionsWorkspace(ctx, devConfig, provider, originalWorkspace, userOptions, log, resolver.WithResolveSubOptions())
}

func ResolveOptions(
	ctx context.Context,
	devConfig *config.Config,
	provider *provider2.ProviderConfig,
	userOptions map[string]string,
	skipRequired bool,
	skipSubOptions bool,
	singleMachine *bool,
	log log.Logger,
) (*config.Config, error) {
	// get binary paths
	binaryPaths, err := binaries.GetBinaries(devConfig.DefaultContext, provider)
	if err != nil {
		return nil, err
	}

	resolverOpts := []resolver.Option{
		resolver.WithResolveGlobal(),
		resolver.WithSkipRequired(skipRequired),
	}
	if !skipSubOptions {
		resolverOpts = append(resolverOpts, resolver.WithResolveSubOptions())
	}

	// create new resolver
	resolve := resolver.New(
		userOptions,
		provider2.Merge(provider2.GetBaseEnvironment(devConfig.DefaultContext, provider.Name), binaryPaths),
		log,
		resolverOpts...,
	)

	// loop and resolve options, as soon as we encounter a new dynamic option it will get filled
	resolvedOptionValues, dynamicOptionDefinitions, err := resolve.Resolve(
		ctx,
		nil,
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
	agentConfig.Dockerless.Image = resolver.ResolveDefaultValue(agentConfig.Dockerless.Image, options)
	agentConfig.Dockerless.Disabled = types.StrBool(resolver.ResolveDefaultValue(string(agentConfig.Dockerless.Disabled), options))
	agentConfig.Dockerless.IgnorePaths = resolver.ResolveDefaultValue(agentConfig.Dockerless.IgnorePaths, options)
	agentConfig.Dockerless.RegistryCache = devConfig.ContextOption(config.ContextOptionRegistryCache)
	agentConfig.Driver = resolver.ResolveDefaultValue(agentConfig.Driver, options)
	agentConfig.Local = types.StrBool(resolver.ResolveDefaultValue(string(agentConfig.Local), options))

	// docker driver
	agentConfig.Docker.Path = resolver.ResolveDefaultValue(agentConfig.Docker.Path, options)
	agentConfig.Docker.Builder = resolver.ResolveDefaultValue(agentConfig.Docker.Builder, options)
	agentConfig.Docker.Install = types.StrBool(resolver.ResolveDefaultValue(string(agentConfig.Docker.Install), options))
	agentConfig.Docker.Env = resolver.ResolveDefaultValues(agentConfig.Docker.Env, options)

	// kubernetes driver
	agentConfig.Kubernetes.KubernetesContext = resolver.ResolveDefaultValue(agentConfig.Kubernetes.KubernetesContext, options)
	agentConfig.Kubernetes.KubernetesConfig = resolver.ResolveDefaultValue(agentConfig.Kubernetes.KubernetesConfig, options)
	agentConfig.Kubernetes.KubernetesNamespace = resolver.ResolveDefaultValue(agentConfig.Kubernetes.KubernetesNamespace, options)
	agentConfig.Kubernetes.Architecture = resolver.ResolveDefaultValue(agentConfig.Kubernetes.Architecture, options)
	agentConfig.Kubernetes.InactivityTimeout = resolver.ResolveDefaultValue(agentConfig.Kubernetes.InactivityTimeout, options)
	agentConfig.Kubernetes.StorageClass = resolver.ResolveDefaultValue(agentConfig.Kubernetes.StorageClass, options)
	agentConfig.Kubernetes.PvcAccessMode = resolver.ResolveDefaultValue(agentConfig.Kubernetes.PvcAccessMode, options)
	agentConfig.Kubernetes.PvcAnnotations = resolver.ResolveDefaultValue(agentConfig.Kubernetes.PvcAnnotations, options)
	agentConfig.Kubernetes.NodeSelector = resolver.ResolveDefaultValue(agentConfig.Kubernetes.NodeSelector, options)
	agentConfig.Kubernetes.Resources = resolver.ResolveDefaultValue(agentConfig.Kubernetes.Resources, options)
	agentConfig.Kubernetes.WorkspaceVolumeMount = resolver.ResolveDefaultValue(agentConfig.Kubernetes.WorkspaceVolumeMount, options)
	agentConfig.Kubernetes.PodManifestTemplate = resolver.ResolveDefaultValue(agentConfig.Kubernetes.PodManifestTemplate, options)
	agentConfig.Kubernetes.Labels = resolver.ResolveDefaultValue(agentConfig.Kubernetes.Labels, options)
	agentConfig.Kubernetes.StrictSecurity = resolver.ResolveDefaultValue(agentConfig.Kubernetes.StrictSecurity, options)
	agentConfig.Kubernetes.CreateNamespace = resolver.ResolveDefaultValue(agentConfig.Kubernetes.CreateNamespace, options)
	agentConfig.Kubernetes.ClusterRole = resolver.ResolveDefaultValue(agentConfig.Kubernetes.ClusterRole, options)
	agentConfig.Kubernetes.ServiceAccount = resolver.ResolveDefaultValue(agentConfig.Kubernetes.ServiceAccount, options)
	agentConfig.Kubernetes.PodTimeout = resolver.ResolveDefaultValue(agentConfig.Kubernetes.PodTimeout, options)
	agentConfig.Kubernetes.KubernetesPullSecretsEnabled = resolver.ResolveDefaultValue(agentConfig.Kubernetes.KubernetesPullSecretsEnabled, options)
	agentConfig.Kubernetes.DiskSize = resolver.ResolveDefaultValue(agentConfig.Kubernetes.DiskSize, options)

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
	if devConfig.ContextOption(config.ContextOptionSSHInjectGitCredentials) != "" {
		agentConfig.InjectGitCredentials = types.StrBool(devConfig.ContextOption(config.ContextOptionSSHInjectGitCredentials))
	}
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
