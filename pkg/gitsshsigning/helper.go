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
	HelperScript string = `#!/bin/bash

devpod agent git-ssh-signature "$@"
`
	HelperScriptPath string = "/usr/local/bin/devpod-ssh-signature"
	GitConfig        string = `
[gpg "ssh"]
	program = devpod-ssh-signature
[gpg]
	format = ssh
`
)

func ConfigureHelper(binaryPath, userName, gitSigningKey string) error {
	helperScriptFile, err := os.Create(HelperScriptPath)
	if err != nil {
		return err
	}
	_, err = helperScriptFile.WriteString(HelperScript)
	if err != nil {
		return err
	}

	err = helperScriptFile.Close()
	if err != nil {
		return err
	}

	err = exec.Command("chmod", "+x", HelperScriptPath).Run()
	if err != nil {
		return err
	}

	homeDir, err := command.GetHome(userName)
	if err != nil {
		return err
	}

	gitConfigPath := filepath.Join(homeDir, ".gitconfig")
	out, err := os.ReadFile(gitConfigPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	config := string(out)
	if !strings.Contains(config, "program = devpod-ssh-signature") {
		content := removeSignatureHelper(config) + GitConfig

		err = os.WriteFile(gitConfigPath, []byte(content), 0600)
		if err != nil {
			return errors.Wrap(err, "write git config")
		}

		err = file.Chown(userName, gitConfigPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func RemoveHelper() error {
	// TODO
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
