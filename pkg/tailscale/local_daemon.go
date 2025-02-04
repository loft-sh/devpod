package tailscale

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/loft-sh/log"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/shell"
	"github.com/loft-sh/devpod/pkg/single"
	"tailscale.com/client/tailscale"
)

// LocalDaemon is a wrapper around the tailscale local client that ensures an userspace
// tailscaled is running and listening for connections on $HOME/.devpod/tailscaled.sock
type LocalDaemon struct {
	lc       *tailscale.LocalClient
	AuthKey  string
	Hostname string
	AuthURL  string
	stateDir string
	execPath string
	logr     log.Logger
	Timeout  time.Duration
}

// UpResponse is the response from tailscale up
type UpResponse struct {
	BackendState string `json:"BackendState"`
}

// NewLocalDaemon returns a LocalDaemon configured to use devpod's userspace tailscale daemon socket
func NewLocalDaemon(authKey, hostname, authUrl, provider, context string, logr log.Logger) *LocalDaemon {
	devpodDir, _ := config.GetConfigDir()
	if context == "" {
		context = "default"
	}
	execPath, err := os.Executable()
	if err != nil {
		execPath = "devpod"
	}
	return &LocalDaemon{
		lc: &tailscale.LocalClient{
			Socket:        fmt.Sprintf("%s/contexts/%s/providers/%s/tailscaled.sock", devpodDir, context, provider),
			UseSocketOnly: true,
		},
		AuthKey:  authKey,
		Hostname: hostname,
		AuthURL:  authUrl,
		stateDir: fmt.Sprintf("%s/contexts/%s/providers/%s/tailscaled/", devpodDir, context, provider),
		execPath: execPath,
		logr:     logr,
		Timeout:  time.Second * 5,
	}
}

// Start conditionally runs the tailscale daemon in the background if not already started
// We then call tailscale up using the CLI to authenticate and connect the daemon to loft's control plane
// Once the daemon is connected our local client is ready to be used
func (d *LocalDaemon) Start(ctx context.Context) error {
	// Ensure userspace daemon is running
	err := single.Single("devpod.tailscaled.pid", func() (*exec.Cmd, error) {
		d.logr.Infof("Starting DevPod Tailscale Daemon ...")
		// todo maybe use shell like below
		return exec.CommandContext(ctx,
			d.execPath,
			"tailscaled",
			"--tun=userspace-networking",
			"--socket", d.lc.Socket,
			"--statedir", d.stateDir,
		), nil
	})
	if err != nil {
		return err
	}

	// Check if we are already connected
	st, err := d.lc.Status(ctx)
	if err == nil && st.BackendState == "Running" {
		return nil
	}

	// Wait for socket to be created by the daemon
	for range 5 {
		if _, err = os.Stat(d.lc.Socket); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Run tailscale up so tailscaled authenticates to the control plane and connects to the tailnet
	// Subsequent calls to Start will now be instant as it will reuse devpodTailscaleSocket
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := os.Environ()
	cmd := fmt.Sprintf(
		"%s tailscale up --auth-key %s --hostname %s --login-server %s --timeout %s --json",
		d.execPath,
		d.AuthKey,
		d.Hostname,
		d.AuthURL,
		d.Timeout,
	)
	err = shell.ExecuteCommandWithShell(ctx, cmd, nil, stdout, stderr, env)
	if err != nil {
		return fmt.Errorf("error running tailscale up: %w, stderr: %s", err, stderr.String())
	}
	// Check response from tailscale up to confirm network is connected
	res := &UpResponse{}
	if err = json.Unmarshal(stdout.Bytes(), &res); err != nil {
		return err
	}
	if res.BackendState != "Running" {
		return fmt.Errorf("tailscale is not running, state: %s", res.BackendState)
	}

	return nil
}

// Stop runs tailscale down using the tailscale CLI via the daemon socket
func (d *LocalDaemon) Stop() {
	cmd := fmt.Sprintf("%s tailscale down", d.execPath)
	shell.ExecuteCommandWithShell(context.Background(), cmd, nil, nil, nil, os.Environ()) // todo fix context
}

// Dial uses the tailscale local client to proxy the connection through the daemon into tailnet
func (d *LocalDaemon) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	// Parse addr as host:port
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		fmt.Println("Error:", err)
		return nil, fmt.Errorf("invalid address: %v", err)
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("invalid port number: %v", err)
	}
	// Use tailscale local client to dial the tailnet address
	return d.lc.DialTCP(ctx, host, uint16(portNum))
}

// LocalClient returns the tailscale local client to the caller
func (t *LocalDaemon) LocalClient() (*tailscale.LocalClient, error) {
	return t.lc, nil
}
