package agent

import (
	"github.com/mitchellh/go-homedir"
	"os"
	"path/filepath"
)

func GetAgentDaemonLogFolder() (string, error) {
	baseFolders := GetBaseFolders()

	// workspace folder
	var lastErr error
	for _, folder := range baseFolders {
		workspaceDir := filepath.Join(folder, "log")

		// check if it already exists
		_, err := os.Stat(workspaceDir)
		if err == nil {
			return workspaceDir, nil
		}

		// create workspace folder
		lastErr = os.MkdirAll(workspaceDir, 0755)
		if lastErr != nil {
			continue
		}

		return workspaceDir, nil
	}

	return "", lastErr
}

func GetBaseFolders() []string {
	baseFolders := []string{}
	homeDir, _ := homedir.Dir()
	if homeDir != "" {
		baseFolders = append(baseFolders, filepath.Join(homeDir, ".devpod", "agent"))
	}

	baseFolders = append(baseFolders, "/home/devpod/.devpod/agent", "/opt/devpod/agent", "/var/devpod/agent")
	return baseFolders
}

func GetAgentWorkspaceContentDir(workspaceDir string) string {
	return filepath.Join(workspaceDir, "content")
}

func GetAgentWorkspaceDir(context, workspaceID string) (string, error) {
	baseFolders := GetBaseFolders()

	// workspace folder
	var lastErr error
	for _, folder := range baseFolders {
		workspaceDir := filepath.Join(folder, "contexts", context, "workspaces", workspaceID)

		// check if it already exists
		_, err := os.Stat(workspaceDir)
		if err == nil {
			return workspaceDir, nil
		}

		// create workspace folder
		lastErr = os.MkdirAll(workspaceDir, 0755)
		if lastErr != nil {
			continue
		}

		return workspaceDir, nil
	}

	return "", lastErr
}
