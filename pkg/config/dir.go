package config

import (
	homedir "github.com/mitchellh/go-homedir"
	"path/filepath"
)

func GetConfigDir() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".devpod")
	return configDir, nil
}
