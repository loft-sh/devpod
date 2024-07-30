package loftconfig

import (
	"os"
	"path/filepath"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider"
)

type LoftConfigResponse struct {
	Config []byte
}

func Read(devPodConfig *config.Config) (*LoftConfigResponse, error) {
	providerDir, err := provider.GetProviderDir(devPodConfig.DefaultContext, "devpod-pro") // TODO: deduplicate with implementation in caller
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
