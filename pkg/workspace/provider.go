package workspace

import (
	"bytes"
	"fmt"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/provider/providerimplementation"
	"github.com/loft-sh/devpod/providers"
	"os"
	"path/filepath"
)

var provideWorkspaceArgErr = fmt.Errorf("please provide a workspace name. E.g. 'devpod up ./my-folder', 'devpod up github.com/my-org/my-repo' or 'devpod up ubuntu'")

type ProviderWithOptions struct {
	Provider provider2.Provider
	Options  map[string]provider2.OptionValue
}

// LoadProviders loads all known providers for the given context and
func LoadProviders(devPodConfig *config.Config, log log.Logger) (*ProviderWithOptions, map[string]*ProviderWithOptions, error) {
	defaultContext := devPodConfig.Contexts[devPodConfig.DefaultContext]
	retProviders, err := LoadAllProviders(devPodConfig, log)
	if err != nil {
		return nil, nil, err
	}

	// get default provider
	if defaultContext.DefaultProvider == "" {
		return nil, nil, fmt.Errorf("no default provider found. Please make sure to run 'devpod use provider'")
	} else if retProviders[defaultContext.DefaultProvider] == nil {
		return nil, nil, fmt.Errorf("couldn't find default provider %s. Please make sure to add the provider via 'devpod add provider'", defaultContext.DefaultProvider)
	}

	return retProviders[defaultContext.DefaultProvider], retProviders, nil
}

func FindProvider(devPodConfig *config.Config, name string, log log.Logger) (*ProviderWithOptions, error) {
	retProviders, err := LoadAllProviders(devPodConfig, log)
	if err != nil {
		return nil, err
	} else if retProviders[name] == nil {
		return nil, fmt.Errorf("couldn't find provider with name %s. Please make sure to add the provider via 'devpod add provider'", name)
	}

	return retProviders[name], nil
}

func LoadAllProviders(devPodConfig *config.Config, log log.Logger) (map[string]*ProviderWithOptions, error) {
	builtInProviders, err := providers.GetBuiltInProviders(log)
	if err != nil {
		return nil, err
	}

	retProviders := map[string]*ProviderWithOptions{}
	for k, p := range builtInProviders {
		retProviders[k] = &ProviderWithOptions{
			Provider: p,
		}
	}

	defaultContext := devPodConfig.Contexts[devPodConfig.DefaultContext]
	for providerName, providerOptions := range defaultContext.Providers {
		if retProviders[providerName] != nil {
			retProviders[providerName].Options = providerOptions.Options
			continue
		}

		// try to load provider config
		providerDir, err := config.GetProviderDir(devPodConfig.DefaultContext, providerName)
		if err != nil {
			log.Errorf("Error retrieving provider directory: %v", err)
			continue
		}

		providerConfigFile := filepath.Join(providerDir, config.ProviderConfigFile)
		contents, err := os.ReadFile(providerConfigFile)
		if err != nil {
			log.Errorf("Error reading provider %s config: %v", providerName, err)
			continue
		}

		providerConfig, err := provider2.ParseProvider(bytes.NewReader(contents))
		if err != nil {
			log.Errorf("Error parsing provider %s config: %v", providerName, err)
			continue
		}

		retProviders[providerName] = &ProviderWithOptions{
			Provider: providerimplementation.NewProvider(providerConfig, log),
			Options:  providerOptions.Options,
		}
	}

	return retProviders, nil
}
