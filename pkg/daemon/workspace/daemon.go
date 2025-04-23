package workspace

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	// RootDir is the directory used by the daemon.
	RootDir                          = "/var/devpod"
	DaemonConfigPath                 = "/var/run/secrets/devpod/daemon_config"
	WorkspaceDaemonConfigExtraEnvVar = "DEVPOD_WORKSPACE_DAEMON_CONFIG"
)

// Daemon holds the config and logger for the daemon.
type Daemon struct {
	Config *DaemonConfig
	Log    log.Logger
}

// NewDaemon creates a new daemon instance.
func NewDaemon() *Daemon {
	return &Daemon{
		Config: &DaemonConfig{},
		Log:    log.NewStreamLogger(os.Stdout, os.Stderr, logrus.InfoLevel),
	}
}

// Run starts the daemon subsystems and waits for an error or termination signal.
func (d *Daemon) Run(c *cobra.Command, args []string) error {
	ctx := c.Context()
	errChan := make(chan error, 4)
	var wg sync.WaitGroup

	if err := d.loadConfig(); err != nil {
		return err
	}

	// Prepare timeout if specified.
	var timeoutDuration time.Duration
	if d.Config.Timeout != "" {
		var err error
		timeoutDuration, err = time.ParseDuration(d.Config.Timeout)
		if err != nil {
			return errors.Wrap(err, "failed to parse timeout duration")
		}
		if timeoutDuration > 0 {
			if err := SetupActivityFile(); err != nil {
				return err
			}
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var tasksStarted bool

	// Start process reaper if running as PID 1.
	if os.Getpid() == 1 {
		wg.Add(1)
		go RunProcessReaper()
	}

	// Start Tailscale networking server.
	if d.shouldRunNetworkServer() {
		tasksStarted = true
		wg.Add(1)
		go RunNetworkServer(ctx, d, errChan, &wg, RootDir)
	}

	// Start timeout monitor.
	if timeoutDuration > 0 {
		tasksStarted = true
		wg.Add(1)
		go RunTimeoutMonitor(ctx, timeoutDuration, errChan, &wg)
	}

	// Start SSH server.
	if d.shouldRunSsh() {
		tasksStarted = true
		wg.Add(1)
		go RunSshServer(ctx, d, errChan, &wg)
	}

	// In case no task is configured, wait indefinitely.
	if !tasksStarted {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ctx.Done()
		}()
	}

	// Listen for OS termination signals.
	go HandleSignals(ctx, errChan)

	// Wait until an error (or termination signal) occurs.
	err := <-errChan
	cancel()
	wg.Wait()

	if err != nil {
		d.Log.Errorf("Daemon error: %v", err)
		os.Exit(1)
	}
	os.Exit(0)
	return nil // Unreachable but needed.
}

// loadConfig loads the daemon configuration from base64-encoded JSON.
// If a CLI-provided timeout exists, it will override the timeout in the config.
func (cmd *Daemon) loadConfig() error {
	// check local file
	encodedCfg := ""
	configBytes, err := os.ReadFile(DaemonConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// check environment variable
			encodedCfg = os.Getenv(config.WorkspaceDaemonConfigExtraEnvVar)
		} else {
			return fmt.Errorf("get daemon config file %s: %w", DaemonConfigPath, err)
		}
	} else {
		encodedCfg = string(configBytes)
	}

	if strings.TrimSpace(encodedCfg) != "" {
		decoded, err := base64.StdEncoding.DecodeString(encodedCfg)
		if err != nil {
			return fmt.Errorf("error decoding daemon config: %w", err)
		}
		var cfg DaemonConfig
		if err = json.Unmarshal(decoded, &cfg); err != nil {
			return fmt.Errorf("error unmarshalling daemon config: %w", err)
		}
		if cmd.Config.Timeout != "" {
			cfg.Timeout = cmd.Config.Timeout
		}
		cmd.Config = &cfg
	}

	return nil
}

// shouldRunNetworkServer returns true if the required platform parameters are present.
func (d *Daemon) shouldRunNetworkServer() bool {
	return d.Config.Platform.AccessKey != "" &&
		d.Config.Platform.PlatformHost != "" &&
		d.Config.Platform.WorkspaceHost != ""
}

// shouldRunSsh returns true if at least one SSH configuration value is provided.
func (d *Daemon) shouldRunSsh() bool {
	return d.Config.Ssh.Workdir != "" || d.Config.Ssh.User != ""
}

// HandleSignals listens for OS termination signals and sends an error through errChan.
func HandleSignals(ctx context.Context, errChan chan<- error) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	select {
	case sig := <-sigChan:
		errChan <- fmt.Errorf("received signal: %v", sig)
	case <-ctx.Done():
	}
}
