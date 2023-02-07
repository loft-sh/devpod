package config

import (
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/pkg/provider"
	homedir "github.com/mitchellh/go-homedir"
	"os"
	"path/filepath"
)

const ProviderConfigFile = "provider.yaml"

const WorkspaceConfigFile = "workspace.json"

func GetConfigDir() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".devpod")
	return configDir, nil
}

func GetWorkspacesDir(context string) (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", context, "workspaces"), nil
}

func GetProviderDir(context, providerName string) (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", context, "providers", providerName), nil
}

func GetWorkspaceDir(context, workspaceID string) (string, error) {
	if workspaceID == "" {
		return "", fmt.Errorf("workspace id is empty")
	}

	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", context, "workspaces", workspaceID), nil
}

func WorkspaceExists(context, workspaceID string) bool {
	workspaceDir, err := GetWorkspaceDir(context, workspaceID)
	if err != nil {
		return false
	}

	_, err = os.Stat(workspaceDir)
	if err != nil {
		return false
	}

	return true
}

func SaveWorkspaceConfig(workspace *provider.Workspace) error {
	workspaceDir, err := GetWorkspaceDir(workspace.Context, workspace.ID)
	if err != nil {
		return err
	}

	err = os.MkdirAll(workspaceDir, 0755)
	if err != nil {
		return err
	}

	workspaceConfigBytes, err := json.Marshal(workspace)
	if err != nil {
		return err
	}

	workspaceConfigFile := filepath.Join(workspaceDir, WorkspaceConfigFile)
	err = os.WriteFile(workspaceConfigFile, workspaceConfigBytes, 0666)
	if err != nil {
		return err
	}

	return nil
}

func LoadWorkspaceConfig(context, workspaceID string) (*provider.Workspace, error) {
	workspaceDir, err := GetWorkspaceDir(context, workspaceID)
	if err != nil {
		return nil, err
	}

	workspaceConfigFile := filepath.Join(workspaceDir, WorkspaceConfigFile)
	workspaceConfigBytes, err := os.ReadFile(workspaceConfigFile)
	if err != nil {
		return nil, err
	}

	workspaceConfig := &provider.Workspace{}
	err = json.Unmarshal(workspaceConfigBytes, workspaceConfig)
	if err != nil {
		return nil, err
	}

	workspaceConfig.Context = context
	workspaceConfig.Origin = workspaceConfigFile
	return workspaceConfig, nil
}
