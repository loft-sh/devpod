package server

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/loft-sh/ssh"
)

func setupAgentListener(sess ssh.Session, reuseSock string) (net.Listener, string, error) {
	// on some systems (like containers) /tmp may not exists, this ensures
	// that we have a compliant directory structure
	err := os.MkdirAll("/tmp", 0o777)
	if err != nil {
		return nil, "", fmt.Errorf("create /tmp dir: %w", err)
	}

	// Check if we should create a "shared" socket to be reused by clients
	// used for browser tunnels such as openvscode, since the IDE itself doesn't create an SSH connection it uses a "backhaul" connection and uses the existing socket
	dir := ""
	if reuseSock != "" {
		dir = filepath.Join(os.TempDir(), fmt.Sprintf("auth-agent-%s", reuseSock))
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			return nil, "", fmt.Errorf("creating SSH_AUTH_SOCK dir in /tmp: %w", err)
		}
	}

	l, tmpDir, err := ssh.NewAgentListener(dir)
	if err != nil {
		return nil, "", fmt.Errorf("new agent listener: %w", err)
	}

	return l, tmpDir, nil
}
