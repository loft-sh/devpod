package crane

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

func IsAvailable() bool {
	_, err := exec.LookPath(BinPath)
	return err == nil
}

func runCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(BinPath, append([]string{command}, args...)...)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %v, error: %w", errBuf.String(), err)
	}

	return outBuf.String(), nil
}

func PullConfigFromSource(configSource string, log log.Logger) (string, error) {
	data, err := runCommand(PullCommand, GitCrane, configSource)
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

	return createContentDirectory(content)
}

func createContentDirectory(content *Content) (string, error) {
	tmpDir, err := os.MkdirTemp("", tmpDirTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	for filename, fileContent := range content.Files {
		filePath := filepath.Join(tmpDir, filename)

		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return "", err
		}

		err := os.WriteFile(filePath, []byte(fileContent), os.ModePerm)
		if err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("failed to write file %s: %w", filename, err)
		}
	}

	return tmpDir, nil
}
