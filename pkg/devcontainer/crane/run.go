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

	GitCrane         = "git"
	EnvironmentCrane = "environment"

	defaultBinName     = "devpod-crane"
	envDevPodCraneName = "DEVPOD_CRANE_NAME"
	tmpDirTemplate     = "devpod-crane-*"
)

type Content struct {
	Files map[string]string `json:"files"`
}

type command struct {
	cmd  string
	args []string
}

func New(cmd string) *command {
	return &command{cmd: cmd}
}

func (c *command) WithFlag(flag, val string) *command {
	c.args = append(c.args, flag, val)
	return c
}

func (c *command) WithArg(arg string) *command {
	c.args = append(c.args, arg)
	return c
}

func (c *command) Run() (string, error) {
	cmd := exec.Command(c.cmd, c.args...)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute command: %v, error: %w", errBuf.String(), err)
	}

	return outBuf.String(), nil
}

// ShouldUse takes CLIOptions and returns true if crane should be used
func ShouldUse(cliOptions *provider2.CLIOptions) bool {
	return IsAvailable() && (cliOptions.DevContainerSource != "" ||
		cliOptions.EnvironmentTemplate != "")
}

// IsAvailable checks if devpod crane is installed in host system
func IsAvailable() bool {
	_, err := exec.LookPath(getBinName())
	return err == nil
}

// PullConfigFromSource pulls devcontainer config from configSource using git crane and returns config path
func PullConfigFromSource(workspaceInfo *provider2.AgentWorkspaceInfo, options *provider2.CLIOptions, log log.Logger) (string, error) {
	var data string
	var err error

	switch {
	case options.EnvironmentTemplate != "":
		data, err = New(PullCommand).WithArg(EnvironmentCrane).WithArg(options.EnvironmentTemplate).Run()
	case options.DevContainerSource != "":
		data, err = New(PullCommand).WithArg(GitCrane).WithArg(options.DevContainerSource).Run()
	default:
		err = fmt.Errorf("failed to pull config from source based on options")
	}
	if err != nil {
		return "", err
	}

	if craneSigningKey != "" {
		data, err = New(DecryptCommand).WithArg(data).WithFlag("--key", craneSigningKey).Run()
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

func getBinName() string {
	if name := os.Getenv(envDevPodCraneName); name != "" {
		return name
	}
	return defaultBinName
}
