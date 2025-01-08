package ssh

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/loft-sh/devpod/pkg/command"
	devsshagent "github.com/loft-sh/devpod/pkg/ssh/agent"
	"github.com/loft-sh/devpod/pkg/util"
	"github.com/loft-sh/log"
	"golang.org/x/crypto/ssh"
)

func AddPrivateKeysToAgent(ctx context.Context, log log.Logger) error {
	if devsshagent.GetSSHAuthSocket() == "" {
		return fmt.Errorf("ssh-agent is not started")
	} else if !command.Exists("ssh-add") {
		return fmt.Errorf("ssh-add couldn't be found")
	}

	privateKeys, err := FindPrivateKeys()
	if err != nil {
		return err
	}

	for _, privateKey := range privateKeys {
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*2)
		log.Debugf("Run ssh-add %s", privateKey)
		out, err := exec.CommandContext(timeoutCtx, "ssh-add", privateKey).CombinedOutput()
		cancel()
		if err != nil {
			log.Debugf("Error adding key %s to agent: %v", privateKey, command.WrapCommandError(out, err))
		}
	}

	return nil
}

func FindPrivateKeys() ([]string, error) {
	homeDir, err := util.UserHomeDir()
	if err != nil {
		return nil, err
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil, err
	}

	privateKeys := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		keyPath := filepath.Join(sshDir, entry.Name())
		out, err := os.ReadFile(keyPath)
		if err == nil {
			_, err = ssh.ParsePrivateKey(out)
			if err == nil {
				privateKeys = append(privateKeys, keyPath)
			}
		}
	}

	return privateKeys, nil
}
