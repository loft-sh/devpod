package gitsshsigning

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/file"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/scanner"
	"github.com/pkg/errors"
)

const (
	HelperScript = `#!/bin/bash

devpod agent git-ssh-signature "$@"
`
	HelperScriptPath  = "/usr/local/bin/devpod-ssh-signature"
	GitConfigTemplate = `
[gpg "ssh"]
	program = devpod-ssh-signature
[gpg]
	format = ssh
[user]
	signingkey = %s
`
)

// ConfigureHelper sets up the Git SSH signing helper script and updates the Git configuration for the specified user.
//
// This function:
// - sets user.signingkey git config
// - creates a wrapper script for calling git-ssh-signature
// - users this script as gpg.ssh.program
// This is needed since git expects `gpg.ssh.program` to be an executable.
func ConfigureHelper(userName, gitSigningKey string, log log.Logger) error {
	log.Debug("Creating helper script")
	if err := createHelperScript(); err != nil {
		return err
	}
	log.Debugf("Helper script created. Making it executable.")
	if err := makeScriptExecutable(); err != nil {
		return err
	}
	log.Debugf("Script executable. Getting config path.")
	gitConfigPath, err := getGitConfigPath(userName)
	if err != nil {
		return err
	}
	log.Debugf("Got config path: %v", gitConfigPath)
	if err := updateGitConfig(gitConfigPath, userName, gitSigningKey); err != nil {
		log.Errorf("Failed updating git configuration: %w", err)
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
	// we do it this way instead of os.Create because we need sudo
	cmd := exec.Command("sudo", "bash", "-c", fmt.Sprintf("echo '%s' > %s", HelperScript, HelperScriptPath))
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func makeScriptExecutable() error {
	return exec.Command("sudo", "chmod", "+x", HelperScriptPath).Run()
}

func getGitConfigPath(userName string) (string, error) {
	homeDir, err := command.GetHome(userName)
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".gitconfig"), nil
}

func updateGitConfig(gitConfigPath, userName, gitSigningKey string) error {
	configContent, err := readGitConfig(gitConfigPath)
	if err != nil {
		return err
	}

	if !strings.Contains(configContent, "program = devpod-ssh-signature") {
		newConfig := fmt.Sprintf(GitConfigTemplate, gitSigningKey)
		newContent := removeSignatureHelper(configContent) + newConfig
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
