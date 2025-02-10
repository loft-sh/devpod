package loftconfig

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/provider"
)

const (
	LoftPlatformConfigFileName = "loft-config.json" // TODO: move somewhere else, replace hardoced strings with usage of this const
)

type LoftConfigRequest struct {
	// Deprecated. Do not use anymore
	Context string
	// Deprecated. Do not use anymore
	Provider string
}

type LoftConfigResponse struct {
	LoftConfig *client.Config
}

func Read(request *LoftConfigRequest) (*LoftConfigResponse, error) {
	loftConfig, err := readConfig(request.Context, request.Provider)
	if err != nil {
		return nil, err
	}

	return &LoftConfigResponse{LoftConfig: loftConfig}, nil
}

func ReadFromWorkspace(workspace *provider.Workspace) (*LoftConfigResponse, error) {
	loftConfig, err := readConfig(workspace.Context, workspace.Provider.Name)
	if err != nil {
		return nil, err
	}

	return &LoftConfigResponse{LoftConfig: loftConfig}, nil
}

func readConfig(contextName string, providerName string) (*client.Config, error) {
	providerDir, err := provider.GetProviderDir(contextName, providerName)
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(providerDir, LoftPlatformConfigFileName)

	// Check if given context and provider have Loft Platform configuration
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// If not just return empty response
		return &client.Config{}, nil
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	loftConfig := &client.Config{}
	err = json.Unmarshal(content, loftConfig)
	if err != nil {
		return nil, err
	}

	return loftConfig, nil
}
