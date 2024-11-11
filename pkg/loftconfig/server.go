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
	Context  string
	Provider string
}

type LoftConfigResponse struct {
	LoftConfig *client.Config
}

func Read(request *LoftConfigRequest) (*LoftConfigResponse, error) {
	providerDir, err := provider.GetProviderDir(request.Context, request.Provider)
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(providerDir, LoftPlatformConfigFileName)

	// Check if given context and provider have Loft Platform configuration
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// If not just return empty response
		return &LoftConfigResponse{}, nil
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

	return &LoftConfigResponse{LoftConfig: loftConfig}, nil
}
