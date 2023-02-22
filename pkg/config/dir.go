package config

import (
	homedir "github.com/mitchellh/go-homedir"
	"path/filepath"
)

const ProviderConfigFile = "provider.yaml"

func GetConfigDir() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".devpod")
	return configDir, nil
}
