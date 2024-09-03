package crane

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
)

var (
	craneSigningKey string
)

const (
	PullCommand    = "pull"
	DecryptCommand = "decrypt"

	GitCrane = "git"

	BinPath = "devpod-crane" // FIXME

	tmpDirTemplate = "devpod-crane-*"
)

type Content struct {
	Files map[string]string `json:"files"`
}

// IsAvailable checks if devpod crane is installed in host system
func IsAvailable() bool {
	_, err := exec.LookPath(BinPath)
	return err == nil
}

func runCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(BinPath, append([]string{command}, args...)...)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute command: %v, error: %w", errBuf.String(), err)
	}

	return outBuf.String(), nil
}

// PullConfigFromSource pulls devcontainer config from configSource using git crane and returns config path
func PullConfigFromSource(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) (string, error) {
	data, err := runCommand(PullCommand, GitCrane, workspaceInfo.CLIOptions.DevContainerSource)
	if err != nil {
		return "", err
	}

	if craneSigningKey != "" {
		data, err = runCommand(DecryptCommand, data, "--key", craneSigningKey)
		if err != nil {
			return "", err
		}
	}

	content := &Content{}
	if err := json.Unmarshal([]byte(data), content); err != nil {
		return "", err
	}

	return writeContentToDirectory(workspaceInfo, content, log)
}

func writeContentToDirectory(workspaceInfo *provider2.AgentWorkspaceInfo, content *Content, _ log.Logger) (string, error) {
	path := workspaceInfo.ContentFolder
	if path == "" {
		path = createContentDirectory()
		if path == "" {
			return path, fmt.Errorf("failed to create temporary directory")
		}
	}
	return storeFilesInDirectory(content, path)
}

func createContentDirectory() string {
	tmpDir, err := os.MkdirTemp("", tmpDirTemplate)
	if err != nil {
		return ""
	}

	return tmpDir
}

func storeFilesInDirectory(content *Content, path string) (string, error) {
	for filename, fileContent := range content.Files {
		filePath := filepath.Join(path, filename)

		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return "", err
		}

		err := os.WriteFile(filePath, []byte(fileContent), os.ModePerm)
		if err != nil {
			os.RemoveAll(path)
			return "", fmt.Errorf("failed to write file %s: %w", filename, err)
		}
	}

	return path, nil
}
