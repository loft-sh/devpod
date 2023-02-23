package clientimplementation

import (
	"fmt"
	client "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
)

func NewWorkspaceClient(parsedConfig *provider.ProviderConfig, workspace *provider.Workspace, log log.Logger) (client.WorkspaceClient, error) {
	// create an interface out of the config
	if parsedConfig.Type == "" || parsedConfig.Type == provider.ProviderTypeServer {
		return NewAgentClient(parsedConfig, workspace, log)
	} else if parsedConfig.Type == provider.ProviderTypeDirect {
		return NewDirectClient(parsedConfig, workspace, log), nil
	}

	// this should never occur and be catched properly in the validate function
	// of the provider config parsing
	return nil, fmt.Errorf("unrecognized provider type " + string(parsedConfig.Type))
}
