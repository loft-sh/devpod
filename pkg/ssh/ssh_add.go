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
	"golang.org/x/term"
)

func AddPrivateKeysToAgent(ctx context.Context, log log.Logger) error {
	if devsshagent.GetSSHAuthSocket() == "" {
		return fmt.Errorf("ssh-agent is not started")
	} else if !command.Exists("ssh-add") {
		return fmt.Errorf("ssh-add couldn't be found")
	}

	privateKeys, err := findPrivateKeys()
	if err != nil {
		return err
	}

	for _, privateKey := range privateKeys {
		log.Debugf("Adding key to SSH Agent: %s", privateKey.path)
		err := addKeyToAgent(ctx, privateKey)
		if err != nil {
			log.Debugf("%v", err)
		}
	}

	return nil
}

type privateKey struct {
	path               string
	requiresPassphrase bool
}

func findPrivateKeys() ([]privateKey, error) {
	homeDir, err := util.UserHomeDir()
	if err != nil {
		return nil, err
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil, err
	}

	keys := []privateKey{}
	passphraseMissingErr := &ssh.PassphraseMissingError{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		keyPath := filepath.Join(sshDir, entry.Name())
		out, err := os.ReadFile(keyPath)
		if err == nil {
			_, err = ssh.ParsePrivateKey(out)
			if err == nil {
				keys = append(keys, privateKey{path: keyPath})
			} else if err.Error() == passphraseMissingErr.Error() {
				// we can check for the passphrase later
				keys = append(keys, privateKey{path: keyPath, requiresPassphrase: true})
			}
		}
	}

	return keys, nil
}

func addKeyToAgent(ctx context.Context, privateKey privateKey) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Minute*5)
	defer cancel()

	// Let users enter the passphrase if the key requires it and we're in an interactive session
	if privateKey.requiresPassphrase && term.IsTerminal(int(os.Stdin.Fd())) {
		cmd := exec.CommandContext(timeoutCtx, "ssh-add", privateKey.path)
		cmd.Stdin = os.Stdin
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Add key %s to agent: %w", privateKey.path, command.WrapCommandError(out, err))
		}

		return nil
	}

	// Normal non-interactive mode
	timeoutCtx, cancel = context.WithTimeout(ctx, time.Second*1)
	defer cancel()
	out, err := exec.CommandContext(timeoutCtx, "ssh-add", privateKey.path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Add key %s to agent: %w", privateKey.path, command.WrapCommandError(out, err))
	}

	return nil
}
