package tailscale

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/loft-sh/devpod/pkg/shell"
	"tailscale.com/client/tailscale"
	"tailscale.com/tsnet"
)

type LocalDaemon struct {
	server   *tsnet.Server
	AuthKey  string
	Hostname string
	AuthURL  string
	Timeout  time.Duration
}

type UpResponse struct {
	BackendState string `json:"BackendState"`
}

func NewLocalDaemon(authKey, hostname string) *LocalDaemon {
	return &LocalDaemon{
		server:  new(tsnet.Server),
		Timeout: time.Second * 5,
	}
}

// LocalClient returns the tailscale API client to the caller
// func (d *LocalDaemon) Up() error {
// 	if d.server == nil {
// 		return fmt.Errorf("tailscale server is not running")
// 	}

// 	_, err := d.server.LocalClient()
// 	if err != nil {
// 		return err
// 	}
// 	// do logic in https://github.com/loft-sh/tailscale/blob/main/cmd/tailscale/cli/up.go with local client
// 	return nil
// }

func (d *LocalDaemon) Start(ctx context.Context, done chan bool) error {
	if d.server == nil {
		return fmt.Errorf("tailscale server is not running")
	}

	lc, err := d.server.LocalClient()
	if err != nil {
		return err
	}

	status, err := lc.Status(ctx)
	if err == nil && status.AuthURL == d.AuthURL {
		// already connected to our loft's tailnet
		return nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := os.Environ()
	cmd := fmt.Sprintf(
		"tailscale up --auth-key %s --hostname %s --login-server %s --timeout %s --json --force-reauth",
		d.AuthKey,
		d.Hostname,
		d.AuthURL,
		d.Timeout,
	)
	err = shell.ExecuteCommandWithShell(ctx, cmd, nil, stdout, stderr, env)
	if err != nil {
		return fmt.Errorf("error running tailscale up: %w, stderr: %s", err, stderr.String())
	}

	res := &UpResponse{}
	if err = json.Unmarshal(stdout.Bytes(), &res); err != nil {
		return err
	}

	if res.BackendState != "Running" {
		return fmt.Errorf("tailscale is not running, state: %s", res.BackendState)
	}

	done <- true

	return nil
}

func (d *LocalDaemon) Stop() error {
	if d.server == nil {
		return fmt.Errorf("tailscale server is not running")
	}

	lc, err := d.server.LocalClient()
	if err != nil {
		return err
	}

	ctx := context.Background() // todo change interface
	_, err = lc.Status(ctx)
	if err != nil {
		// already not connected
		return nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := os.Environ()
	cmd := "tailscale down"
	err = shell.ExecuteCommandWithShell(ctx, cmd, nil, stdout, stderr, env)
	if err != nil {
		return fmt.Errorf("error running tailscale down: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func (d *LocalDaemon) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	// if d.server == nil {
	// 	return nil, fmt.Errorf("tailscale server is not running")
	// }
	// return d.server.Dial(ctx, network, addr)
	// since we are not in userspace you can simply connect via network
	return net.Dial(network, addr)
}

// LocalClient returns the tailscale API client to the caller
func (t *LocalDaemon) LocalClient() (*tailscale.LocalClient, error) {
	if t.server == nil {
		return nil, fmt.Errorf("tailscale server is not running")
	}
	return t.server.LocalClient()
}
