package gitsshsigning

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/file"
	"github.com/loft-sh/log/scanner"
	"github.com/pkg/errors"
)

const (
	HelperScript = `#!/bin/bash

devpod agent git-ssh-signature "$@"
`
	HelperScriptPath = "/usr/local/bin/devpod-ssh-signature"
	GitConfig        = `
[gpg "ssh"]
	program = devpod-ssh-signature
[gpg]
	format = ssh
`
)

// ConfigureHelper sets up the git SSH signing helper script and updates the git configuration for the specified user.
func ConfigureHelper(binaryPath, userName, gitSigningKey string) error {
	if err := createHelperScript(); err != nil {
		return err
	}

	if err := makeScriptExecutable(); err != nil {
		return err
	}

	gitConfigPath, err := getGitConfigPath(userName)
	if err != nil {
		return err
	}

	if err := updateGitConfig(gitConfigPath, userName); err != nil {
		return err
	}

	return nil
}

// RemoveHelper removes the git SSH signing helper script and any related configuration.
func RemoveHelper(userName string) error {
	if err := os.Remove(HelperScriptPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	gitConfigPath, err := getGitConfigPath(userName)
	if err != nil {
		return err
	}

	if err := removeGitConfigHelper(gitConfigPath, userName); err != nil {
		return err
	}

	return nil
}

func createHelperScript() error {
	helperScriptFile, err := os.Create(HelperScriptPath)
	if err != nil {
		return err
	}
	defer helperScriptFile.Close()

	_, err = helperScriptFile.WriteString(HelperScript)
	return err
}

func makeScriptExecutable() error {
	return exec.Command("chmod", "+x", HelperScriptPath).Run()
}

func getGitConfigPath(userName string) (string, error) {
	homeDir, err := command.GetHome(userName)
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".gitconfig"), nil
}

func updateGitConfig(gitConfigPath, userName string) error {
	configContent, err := readGitConfig(gitConfigPath)
	if err != nil {
		return err
	}

	if !strings.Contains(configContent, "program = devpod-ssh-signature") {
		newContent := removeSignatureHelper(configContent) + GitConfig
		if err := writeGitConfig(gitConfigPath, newContent, userName); err != nil {
			return err
		}
	}

	return nil
}

func readGitConfig(gitConfigPath string) (string, error) {
	out, err := os.ReadFile(gitConfigPath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	return string(out), nil
}

func writeGitConfig(gitConfigPath, content, userName string) error {
	if err := os.WriteFile(gitConfigPath, []byte(content), 0600); err != nil {
		return errors.Wrap(err, "write git config")
	}
	return file.Chown(userName, gitConfigPath)
}

func removeGitConfigHelper(gitConfigPath, userName string) error {
	configContent, err := readGitConfig(gitConfigPath)
	if err != nil {
		return err
	}

	newContent := removeSignatureHelper(configContent)
	if err := writeGitConfig(gitConfigPath, newContent, userName); err != nil {
		return err
	}

	return nil
}

func removeSignatureHelper(content string) string {
	scan := scanner.NewScanner(strings.NewReader(content))
	isGpgSetup := false
	out := []string{}

	for scan.Scan() {
		line := scan.Text()
		if strings.TrimSpace(line) == "[gpg \"ssh\"]" {
			isGpgSetup = true
			continue
		} else if strings.TrimSpace(line) == "[gpg]" {
			isGpgSetup = true
		} else if isGpgSetup {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 0 && trimmed[0] == '[' {
				isGpgSetup = false
			} else {
				continue
			}
		}
		out = append(out, line)
	}

	return strings.Join(out, "\n")
}
