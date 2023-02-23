package provider

import (
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/pkg/config"
	"os"
	"path/filepath"
)

const WorkspaceConfigFile = "workspace.json"

const ServerConfigFile = "server.json"

func GetServersDir(context string) (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", context, "servers"), nil
}

func GetWorkspacesDir(context string) (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", context, "workspaces"), nil
}

func GetProviderDir(context, providerName string) (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", context, "providers", providerName), nil
}

func GetProviderBinariesDir(context, providerName string) (string, error) {
	providerDir, err := GetProviderDir(context, providerName)
	if err != nil {
		return "", err
	}

	return filepath.Join(providerDir, "binaries"), nil
}

func GetServerDir(context, serverID string) (string, error) {
	if serverID == "" {
		return "", fmt.Errorf("server id is empty")
	}

	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "contexts", context, "servers", serverID), nil
}

func GetWorkspaceDir(context, workspaceID string) (string, error) {
	if workspaceID == "" {
		return "", fmt.Errorf("workspace id is empty")
	}

	configDir, err := config.GetConfigDir()
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

func SaveWorkspaceConfig(workspace *Workspace) error {
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

func SaveServerConfig(server *Server) error {
	serverDir, err := GetServerDir(server.Context, server.ID)
	if err != nil {
		return err
	}

	err = os.MkdirAll(serverDir, 0755)
	if err != nil {
		return err
	}

	serverConfigBytes, err := json.Marshal(server)
	if err != nil {
		return err
	}

	serverConfigFile := filepath.Join(serverDir, ServerConfigFile)
	err = os.WriteFile(serverConfigFile, serverConfigBytes, 0666)
	if err != nil {
		return err
	}

	return nil
}

func ServerExists(context, serverID string) bool {
	serverDir, err := GetServerDir(context, serverID)
	if err != nil {
		return false
	}

	_, err = os.Stat(serverDir)
	if err != nil {
		return false
	}

	return true
}

func LoadServerConfig(context, serverID string) (*Server, error) {
	serverDir, err := GetServerDir(context, serverID)
	if err != nil {
		return nil, err
	}

	serverConfigFile := filepath.Join(serverDir, ServerConfigFile)
	serverConfigBytes, err := os.ReadFile(serverConfigFile)
	if err != nil {
		return nil, err
	}

	serverConfig := &Server{}
	err = json.Unmarshal(serverConfigBytes, serverConfig)
	if err != nil {
		return nil, err
	}

	serverConfig.Context = context
	serverConfig.Origin = serverConfigFile
	return serverConfig, nil
}

func LoadWorkspaceConfig(context, workspaceID string) (*Workspace, error) {
	workspaceDir, err := GetWorkspaceDir(context, workspaceID)
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

	workspaceConfig.Context = context
	workspaceConfig.Origin = workspaceConfigFile
	return workspaceConfig, nil
}
