package config

import (
	"encoding/json"
	"fmt"
	homedir "github.com/mitchellh/go-homedir"
	"os"
	"path/filepath"
)

const DefaultContext = "default"

const WorkspaceConfigFile = "workspace.json"

func GetConfigDir() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".devpod")
	return configDir, nil
}

func GetWorkspacesDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", DefaultContext, "workspaces"), nil
}

func GetWorkspaceDir(workspaceID string) (string, error) {
	if workspaceID == "" {
		return "", fmt.Errorf("workspace id is empty")
	}

	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", DefaultContext, "workspaces", workspaceID), nil
}

func WorkspaceExists(id string) bool {
	workspaceDir, err := GetWorkspaceDir(id)
	if err != nil {
		return false
	}

	_, err = os.Stat(workspaceDir)
	if err != nil {
		return false
	}

	return true
}

func SaveWorkspaceConfig(workspace *Workspace) error {
	workspaceDir, err := GetWorkspaceDir(workspace.ID)
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

func LoadWorkspaceConfig(workspaceID string) (*Workspace, error) {
	workspaceDir, err := GetWorkspaceDir(workspaceID)
	if err != nil {
		return nil, err
	}

	workspaceConfigFile := filepath.Join(workspaceDir, WorkspaceConfigFile)
	workspaceConfigBytes, err := os.ReadFile(workspaceConfigFile)
	if err != nil {
		return nil, err
	}

	workspaceConfig := &Workspace{}
	err = json.Unmarshal(workspaceConfigBytes, workspaceConfig)
	if err != nil {
		return nil, err
	}

	workspaceConfig.Origin = workspaceConfigFile
	return workspaceConfig, nil
}
