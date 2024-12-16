package tunnel

import (
	"context"
	"io"
	"os"

	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/pkg/errors"
)

// Tunnel defines the function to create an "outer" tunnel
type Tunnel func(ctx context.Context, stdin io.Reader, stdout io.Writer) error

// NewTunnel creates a tunnel to the devcontainer using generic functions to establish the "outer" and "inner" tunnel, used by proxy clients
// Here the tunnel will be an SSH connection with it's STDIO as arguments and the handler will be the function to execute the command
// using the connected SSH client.
func NewTunnel(ctx context.Context, tunnel Tunnel, handler Handler) error {
	// create context
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// create readers
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer stdoutWriter.Close()
	defer stdinWriter.Close()

	// start ssh proxy
	outerTunnelChan := make(chan error, 1)
	go func() {
		outerTunnelChan <- tunnel(ctx, stdinReader, stdoutWriter)
	}()

	// start ssh client as root / default user
	innerTunnelChan := make(chan error, 1)
	go func() {
		sshClient, err := devssh.StdioClient(stdoutReader, stdinWriter, false)
		if err != nil {
			innerTunnelChan <- err
			return
		}
		defer sshClient.Close()
		defer cancel()

		// start ssh tunnel
		innerTunnelChan <- handler(cancelCtx, sshClient)
	}()

	// wait for result
	select {
	case err := <-innerTunnelChan:
		return errors.Wrap(err, "inner tunnel")
	case err := <-outerTunnelChan:
		return errors.Wrap(err, "outer tunnel")
	}
}
