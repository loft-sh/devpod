package loftconfig

import (
	"os"
	"path/filepath"

	"github.com/loft-sh/devpod/pkg/provider"
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

	configPath := filepath.Join(providerDir, "loft-config.json")

	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	return &LoftConfigResponse{Config: content}, nil
}
