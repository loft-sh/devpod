package providerimplementation

import "github.com/loft-sh/devpod/pkg/provider"

func NewProvider(parsedConfig *provider.ProviderConfig) provider.Provider {
	// create an interface out of the config
	if parsedConfig.Type == "" || parsedConfig.Type == provider.ProviderTypeServer {
		return NewServerProvider(parsedConfig)
	} else if parsedConfig.Type == provider.ProviderTypeWorkspace {
		return NewWorkspaceProvider(parsedConfig)
	}

	// this should never occur and be catched properly in the validate function
	// of the provider config parsing
	panic("unrecognized provider type " + parsedConfig.Type)
}
