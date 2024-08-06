package loftconfig

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/loft-sh/devpod/pkg/loft/client"
	"github.com/loft-sh/devpod/pkg/provider"
)

func LoftConfigPath(context, providerName string) (string, error) {
	providerDir, err := provider.GetProviderDir(context, providerName)
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(providerDir, "loft-config.json")

	return configPath, nil
}

func StoreLoftConfig(configResponse *LoftConfigResponse, configPath string) error {
	config := &client.Config{}
	err := json.Unmarshal(configResponse.LoftConfig, config)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(configPath), 0o755)
	if err != nil {
		return err
	}

	out, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, out, 0o660)
}
