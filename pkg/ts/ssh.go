package ts

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/log"
	"golang.org/x/crypto/ssh"
	tsClient "tailscale.com/client/tailscale"
)

func WaitForSSHClient(ctx context.Context, lc *tsClient.LocalClient, host string, port int, user string, log log.Logger) (*ssh.Client, error) {
	deadline := time.Now().Add(10 * time.Second)

	var (
		c   *ssh.Client
		err error
	)
	log.Debugf("Attempting to establish SSH connection with %s as user %s", host, user)
	for time.Now().Before(deadline) {
		c, err = newSSHClient(ctx, lc, host, port, user)
		if err == nil {
			return c, nil
		}
		select {
		case <-ctx.Done():
			return c, err
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
	log.Debugf("Failed to establish SSH connection %v", err)

	return c, err
}

func newSSHClient(ctx context.Context, lc *tsClient.LocalClient, host string, port int, user string) (*ssh.Client, error) {
	conn, err := lc.DialTCP(ctx, host, uint16(port))
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", host, err)
	}

	clientConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{}, // The SSH server is only reachable through the tailnet
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	serverAddress := fmt.Sprintf("%s:%d", host, port)
	sshConn, channels, requests, err := ssh.NewClientConn(conn, serverAddress, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("establish SSH connection: %w", err)
	}

	return ssh.NewClient(sshConn, channels, requests), nil
}
