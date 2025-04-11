package container

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/loft-sh/devpod/pkg/agent"
	agentd "github.com/loft-sh/devpod/pkg/daemon/agent"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	RootDir          = "/var/devpod"
	DaemonConfigPath = "/var/run/secrets/devpod/daemon_config"
)

type DaemonCmd struct {
	Config *agentd.DaemonConfig
	Log    log.Logger
}

// NewDaemonCmd creates the merged daemon command.
func NewDaemonCmd() *cobra.Command {
	cmd := &DaemonCmd{
		Config: &agentd.DaemonConfig{},
		Log:    log.NewStreamLogger(os.Stdout, os.Stderr, logrus.InfoLevel),
	}
	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Starts the DevPod network daemon, SSH server and monitors container activity if timeout is set",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}
	daemonCmd.Flags().StringVar(&cmd.Config.Timeout, "timeout", "", "The timeout to stop the container after")
	return daemonCmd
}

func (cmd *DaemonCmd) Run(c *cobra.Command, args []string) error {
	ctx := c.Context()
	errChan := make(chan error, 4)
	var wg sync.WaitGroup

	if err := cmd.loadConfig(); err != nil {
		return err
	}

	// Prepare timeout if specified.
	var timeoutDuration time.Duration
	if cmd.Config.Timeout != "" {
		var err error
		timeoutDuration, err = time.ParseDuration(cmd.Config.Timeout)
		if err != nil {
			return errors.Wrap(err, "failed to parse timeout duration")
		}
		if timeoutDuration > 0 {
			if err := setupActivityFile(); err != nil {
				return err
			}
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var tasksStarted bool

	// Start process reaper.
	if os.Getpid() == 1 {
		wg.Add(1)
		go runReaper(ctx, errChan, &wg)
	}

	// Start Tailscale networking server.
	if cmd.shouldRunNetworkServer() {
		tasksStarted = true
		wg.Add(1)
		go runNetworkServer(ctx, cmd, errChan, &wg)
	}

	// Start timeout monitor.
	if timeoutDuration > 0 {
		tasksStarted = true
		wg.Add(1)
		go runTimeoutMonitor(ctx, timeoutDuration, errChan, &wg)
	}

	// Start ssh server.
	if cmd.shouldRunSsh() {
		tasksStarted = true
		wg.Add(1)
		go runSshServer(ctx, cmd, errChan, &wg)
	}

	// In case no task is configured, just wait indefinitely.
	if !tasksStarted {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ctx.Done()
		}()
	}

	// Listen for OS termination signals.
	go handleSignals(ctx, errChan)

	// Wait until an error (or termination signal) occurs.
	err := <-errChan
	cancel()
	wg.Wait()

	if err != nil {
		cmd.Log.Errorf("Daemon error: %v", err)
		os.Exit(1)
	}
	os.Exit(0)
	return nil // Unreachable but needed.
}

// loadConfig loads the daemon configuration from base64-encoded JSON.
// If a CLI-provided timeout exists, it will override the timeout in the config.
func (cmd *DaemonCmd) loadConfig() error {
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
		var cfg agentd.DaemonConfig
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
func (cmd *DaemonCmd) shouldRunNetworkServer() bool {
	return cmd.Config.Platform.AccessKey != "" &&
		cmd.Config.Platform.PlatformHost != "" &&
		cmd.Config.Platform.WorkspaceHost != ""
}

// shouldRunSsh returns true if at least one SSH configuration value is provided.
func (cmd *DaemonCmd) shouldRunSsh() bool {
	return cmd.Config.Ssh.Workdir != "" || cmd.Config.Ssh.User != ""
}

// setupActivityFile creates and sets permissions on the container activity file.
func setupActivityFile() error {
	if err := os.WriteFile(agent.ContainerActivityFile, nil, 0777); err != nil {
		return err
	}
	return os.Chmod(agent.ContainerActivityFile, 0777)
}

// runReaper starts the process reaper and waits for context cancellation.
func runReaper(ctx context.Context, errChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()
	agentd.RunProcessReaper()
	<-ctx.Done()
}

// runTimeoutMonitor monitors the activity file and signals an error if the timeout is exceeded.
func runTimeoutMonitor(ctx context.Context, duration time.Duration, errChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stat, err := os.Stat(agent.ContainerActivityFile)
			if err != nil {
				continue
			}
			if !stat.ModTime().Add(duration).After(time.Now()) {
				errChan <- errors.New("timeout reached, terminating daemon")
				return
			}
		}
	}
}

// runNetworkServer starts the network server.
func runNetworkServer(ctx context.Context, cmd *DaemonCmd, errChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()
	if err := os.MkdirAll(RootDir, os.ModePerm); err != nil {
		errChan <- err
		return
	}
	logger := initLogging()
	config := client.NewConfig()
	config.AccessKey = cmd.Config.Platform.AccessKey
	config.Host = "https://" + cmd.Config.Platform.PlatformHost
	config.Insecure = true
	baseClient := client.NewClientFromConfig(config)
	if err := baseClient.RefreshSelf(ctx); err != nil {
		errChan <- fmt.Errorf("failed to refresh client: %w", err)
		return
	}
	tsServer := ts.NewWorkspaceServer(&ts.WorkspaceServerConfig{
		AccessKey:     cmd.Config.Platform.AccessKey,
		PlatformHost:  ts.RemoveProtocol(cmd.Config.Platform.PlatformHost),
		WorkspaceHost: cmd.Config.Platform.WorkspaceHost,
		Client:        baseClient,
		RootDir:       RootDir,
		LogF: func(format string, args ...interface{}) {
			logger.Infof(format, args...)
		},
	}, logger)
	if err := tsServer.Start(ctx); err != nil {
		errChan <- fmt.Errorf("network server: %w", err)
	}
}

// runSshServer starts the SSH server.
func runSshServer(ctx context.Context, cmd *DaemonCmd, errChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()
	binaryPath, err := os.Executable()
	if err != nil {
		errChan <- err
		return
	}

	args := []string{"agent", "container", "ssh-server"}
	if cmd.Config.Ssh.Workdir != "" {
		args = append(args, "--workdir", cmd.Config.Ssh.Workdir)
	}
	if cmd.Config.Ssh.User != "" {
		args = append(args, "--remote-user", cmd.Config.Ssh.User)
	}

	sshCmd := exec.Command(binaryPath, args...)
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	if err := sshCmd.Start(); err != nil {
		errChan <- fmt.Errorf("failed to start SSH server: %w", err)
		return
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			if sshCmd.Process != nil {
				if err := sshCmd.Process.Signal(syscall.SIGTERM); err != nil {
					errChan <- fmt.Errorf("failed to send SIGTERM to SSH server: %w", err)
				}
			}
		case <-done:
		}
	}()

	if err := sshCmd.Wait(); err != nil {
		errChan <- fmt.Errorf("SSH server exited abnormally: %w", err)
		close(done)
		return
	}
	close(done)
}

// handleSignals listens for OS termination signals and sends an error through errChan.
func handleSignals(ctx context.Context, errChan chan<- error) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	select {
	case sig := <-sigChan:
		errChan <- fmt.Errorf("received signal: %v", sig)
	case <-ctx.Done():
	}
}

// initLogging initializes logging and returns a combined logger.
func initLogging() log.Logger {
	return log.NewStdoutLogger(nil, os.Stdout, os.Stderr, logrus.InfoLevel)
}
