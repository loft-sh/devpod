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

func GetSnapshotDir(provider, workspaceID string) (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "providers", provider, "snapshots", workspaceID), nil
}

func GetWorkspaceDir(provider, workspaceID string) (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "providers", provider, "workspaces", workspaceID), nil
}
