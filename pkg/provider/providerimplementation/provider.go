package providerimplementation

import (
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
)

func NewProvider(parsedConfig *provider.ProviderConfig, log log.Logger) provider.Provider {
	// create an interface out of the config
	if parsedConfig.Type == "" || parsedConfig.Type == provider.ProviderTypeServer {
		return NewServerProvider(parsedConfig, log)
	} else if parsedConfig.Type == provider.ProviderTypeWorkspace {
		return NewWorkspaceProvider(parsedConfig, log)
	}

	// this should never occur and be catched properly in the validate function
	// of the provider config parsing
	panic("unrecognized provider type " + parsedConfig.Type)
}
