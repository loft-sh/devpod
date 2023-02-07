package agent

import (
	"github.com/mitchellh/go-homedir"
	"os"
	"path/filepath"
)

func getBaseFolders() []string {
	baseFolders := []string{}
	homeDir, _ := homedir.Dir()
	if homeDir != "" {
		baseFolders = append(baseFolders, homeDir)
	}

	baseFolders = append(baseFolders, "/home/devpod", "/opt", "/var")
	return baseFolders
}

func GetAgentWorkspaceDir(context, workspaceID string) (string, error) {
	baseFolders := getBaseFolders()

	// workspace folder
	var lastErr error
	for _, folder := range baseFolders {
		workspaceDir := filepath.Join(folder, ".devpod", "contexts", context, "workspaces", workspaceID)

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
