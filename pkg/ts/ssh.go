package ts

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/loft-sh/log"
	"golang.org/x/crypto/ssh"
)

type Dialer func(ctx context.Context, network, address string) (net.Conn, error)

func WaitForSSHClient(ctx context.Context, dialer Dialer, network, address string, user string, timeout time.Duration, log log.Logger) (*ssh.Client, error) {
	deadline := time.Now().Add(timeout)

	var (
		c   *ssh.Client
		err error
	)
	log.Debugf("Attempting to establish SSH connection with %s as user %s", address, user)
	for time.Now().Before(deadline) {
		c, err = newSSHClient(ctx, dialer, network, address, user)
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

func newSSHClient(ctx context.Context, dialer Dialer, network, address string, user string) (*ssh.Client, error) {
	conn, err := dialer(ctx, network, address)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", address, err)
	}

	clientConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{}, // The SSH server is only reachable through the tailnet
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshConn, channels, requests, err := ssh.NewClientConn(conn, address, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("establish SSH connection: %w", err)
	}

	return ssh.NewClient(sshConn, channels, requests), nil
}
