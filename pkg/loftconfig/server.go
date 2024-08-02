package loftconfig

import (
	"os"
	"path/filepath"

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
	Config []byte
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
		return &LoftConfigResponse{Config: []byte{}}, nil
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	return &LoftConfigResponse{Config: content}, nil
}
