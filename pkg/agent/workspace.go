package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/mitchellh/go-homedir"
)

var extraSearchLocations = []string{"/home/devpod/.devpod/agent", "/opt/devpod/agent", "/var/lib/devpod/agent", "/var/devpod/agent"}

var ErrFindAgentHomeFolder = fmt.Errorf("couldn't find devpod home directory")

func GetAgentDaemonLogFolder(agentFolder string) (string, error) {
	return FindAgentHomeFolder(agentFolder)
}

func findDir(agentFolder string, validate func(path string) bool) string {
	// get agent folder
	if agentFolder != "" {
		if !validate(agentFolder) {
			return ""
		}

		return agentFolder
	}

	// check environment
	homeFolder := os.Getenv(config.DEVPOD_HOME)
	if homeFolder != "" && validate(homeFolder) {
		return homeFolder
	}

	// check home folder first
	homeDir, _ := homedir.Dir()
	if homeDir != "" {
		homeDir = filepath.Join(homeDir, ".devpod", "agent")
		if validate(homeDir) {
			return homeDir
		}
	}

	// check root folder
	homeDir, _ = command.GetHome("root")
	if homeDir != "" {
		homeDir = filepath.Join(homeDir, ".devpod", "agent")
		if validate(homeDir) {
			return homeDir
		}
	}

	// check current directory
	execDir, _ := os.Executable()
	if execDir != "" {
		execDir = filepath.Join(filepath.Dir(execDir), "agent")
		if validate(execDir) {
			return execDir
		}
	}

	// check other folders
	for _, dir := range extraSearchLocations {
		if validate(dir) {
			return dir
		}
	}

	return ""
}

func FindAgentHomeFolder(agentFolder string) (string, error) {
	homeDir := findDir(agentFolder, isDevPodHome)
	if homeDir != "" {
		return homeDir, nil
	}

	return "", ErrFindAgentHomeFolder
}

func isDevPodHome(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "contexts"))
	return err == nil
}

func PrepareAgentHomeFolder(agentFolder string) (string, error) {
	// try to find agent home folder first
	homeFolder, err := FindAgentHomeFolder(agentFolder)
	if err == nil {
		return homeFolder, nil
	}

	// try to find an executable directory
	homeDir := findDir(agentFolder, func(path string) bool {
		ok, _ := isDirExecutable(path)
		return ok
	})
	if homeDir != "" {
		return homeDir, nil
	}

	// check if agentFolder is set
	if agentFolder != "" {
		_, err := isDirExecutable(agentFolder)
		return "", err
	}

	// return generic error here
	return "", fmt.Errorf("couldn't find an executable directory")
}

func isDirExecutable(dir string) (bool, error) {
	if !filepath.IsAbs(dir) {
		var err error
		dir, err = filepath.Abs(dir)
		if err != nil {
			return false, err
		}
	}

	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return false, err
	}

	testFile := filepath.Join(dir, "devpod_test.sh")
	err = os.WriteFile(testFile, []byte(`#!/bin/sh
echo DevPod
`), 0755)
	if err != nil {
		return false, err
	}
	defer os.Remove(testFile)
	if runtime.GOOS != "linux" {
		return true, nil
	}

	// try to execute
	out, err := exec.Command(testFile).Output()
	if err != nil {
		return false, err
	} else if strings.TrimSpace(string(out)) != "DevPod" {
		return false, fmt.Errorf("received %s, expected DevPod", strings.TrimSpace(string(out)))
	}

	return true, nil
}

func GetAgentWorkspaceContentDir(workspaceDir string) string {
	return filepath.Join(workspaceDir, "content")
}

func GetAgentBinariesDir(agentFolder, context, workspaceID string) (string, error) {
	homeFolder, err := FindAgentHomeFolder(agentFolder)
	if err != nil {
		return "", err
	}
	if context == "" {
		context = config.DefaultContext
	}

	// workspace folder
	workspaceDir := filepath.Join(homeFolder, "contexts", context, "workspaces", workspaceID)

	// check if it already exists
	_, err = os.Stat(workspaceDir)
	if err == nil {
		return filepath.Join(workspaceDir, "binaries"), nil
	}

	return "", os.ErrNotExist
}

func GetAgentWorkspaceDir(agentFolder, context, workspaceID string) (string, error) {
	homeFolder, err := FindAgentHomeFolder(agentFolder)
	if err != nil {
		return "", err
	}
	if context == "" {
		context = config.DefaultContext
	}

	// workspace folder
	workspaceDir := filepath.Join(homeFolder, "contexts", context, "workspaces", workspaceID)

	// check if it already exists
	_, err = os.Stat(workspaceDir)
	if err == nil {
		return workspaceDir, nil
	}

	return "", os.ErrNotExist
}

func CreateAgentWorkspaceDir(agentFolder, context, workspaceID string) (string, error) {
	homeFolder, err := PrepareAgentHomeFolder(agentFolder)
	if err != nil {
		return "", err
	}

	// workspace folder
	workspaceDir := filepath.Join(homeFolder, "contexts", context, "workspaces", workspaceID)

	// create workspace folder
	err = os.MkdirAll(workspaceDir, 0755)
	if err != nil {
		return "", err
	}

	return workspaceDir, nil
}
