package gitsshsigning

import (
	"os/exec"
	"strings"
)

const (
	GPGFormatConfigKey       = "gpg.format"
	UsersSigningKeyConfigKey = "user.signingkey"
	GPGFormatSSH             = "ssh"
)

// ExtractGitConfiguration is used for extracting values from users local .gitconfig
// that are needed to setup devpod-ssh-signature helper inside the workspace.
func ExtractGitConfiguration() (string, string, error) {
	format, err := readGitConfigValue(GPGFormatConfigKey)
	if err != nil {
		return "", "", err
	}

	signingKey, err := readGitConfigValue(UsersSigningKeyConfigKey)
	if err != nil {
		return "", "", err
	}

	return format, signingKey, nil
}

func readGitConfigValue(key string) (string, error) {
	cmd := exec.Command("git", "config", "--get", key)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
