package crane

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/loft-sh/log"
)

const (
	PullCommand = "pull"
	GitSource   = "git"
	BinPath     = "devpod-crane" // FIXME

	tmpDirTemplate = "devpod-crane-*"
)

type DevContainerConfigPath string

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

func PullConfigFromSource(configSource string, log log.Logger) (DevContainerConfigPath, error) {
	out, err := runCommand(PullCommand, GitSource, configSource)
	if err != nil {
		return "", err
	}

	decryptedData, err := decrypt(out, "")
	if err != nil {
		return "", err
	}

	content := &Content{}
	if err := json.Unmarshal(decryptedData, content); err != nil {
		return "", err
	}

	tmpDir, err := os.MkdirTemp("", tmpDirTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	devcontainerDir := filepath.Join(tmpDir, ".devcontainer") // FIXME

	if err := os.Mkdir(devcontainerDir, 0777); err != nil {
		return "", err
	}

	for filename, fileContent := range content.Files {
		filePath := filepath.Join(devcontainerDir, filename)
		err := os.WriteFile(filePath, []byte(fileContent), 0777)
		if err != nil {
			log.Debugf("Failed creating file %v : %w", filename, err)
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("failed to write file %s: %w", filename, err)
		}
	}

	return DevContainerConfigPath(tmpDir), nil
}

func decrypt(encryptedData, key string) ([]byte, error) {
	if key == "" { // FIXME
		return []byte(encryptedData), nil
	}

	ciphertext, err := base64.URLEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return ciphertext, nil
}
