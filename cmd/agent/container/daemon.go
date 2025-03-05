package container

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	devpodlog "github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// DaemonCmd holds the merged command flags.
type DaemonCmd struct {
	Timeout       string
	AccessKey     string
	PlatformHost  string
	WorkspaceHost string
}

// NewDaemonCmd creates the new merged daemon command.
func NewDaemonCmd() *cobra.Command {
	cmd := &DaemonCmd{}
	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Starts the DevPod network daemon and monitors container activity if timeout is set",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}
	daemonCmd.Flags().StringVar(&cmd.Timeout, "timeout", "", "The timeout to stop the container after")
	daemonCmd.Flags().StringVar(&cmd.AccessKey, "access-key", "", "Access key for the platform")
	daemonCmd.Flags().StringVar(&cmd.PlatformHost, "platform-host", "", "Platform host")
	daemonCmd.Flags().StringVar(&cmd.WorkspaceHost, "workspace-host", "", "Workspace host")
	return daemonCmd
}

// Run runs the command logic
func (cmd *DaemonCmd) Run(c *cobra.Command, args []string) error {
	// Load configuration from flags and/or environment variables.
	cmd.loadConfig()

	ctx := c.Context()

	// Prepare timeout if provided.
	var timeoutDuration time.Duration
	if cmd.Timeout != "" {
		var err error
		timeoutDuration, err = time.ParseDuration(cmd.Timeout)
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

	// Start network and timeout tasks if configured.
	errChan := make(chan error, 2)
	var wg sync.WaitGroup
	tasksStarted := false

	if cmd.shouldRunNetworkServer() {
		tasksStarted = true
		wg.Add(1)
		go runNetworkServer(ctx, cmd, errChan, &wg)
	}
	if timeoutDuration > 0 {
		tasksStarted = true
		wg.Add(1)
		go runTimeoutMonitor(ctx, timeoutDuration, errChan, &wg)
	}
	if !tasksStarted {
		// Block indefinitely if no task is configured.
		select {}
	}

	// Listen for OS signals.
	go handleSignals(ctx, errChan)

	// Wait until an error (or termination signal) occurs.
	err := <-errChan
	cancel()
	wg.Wait()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Daemon error: %v\n", err) // Logging the error for visibility
		os.Exit(1)
	}

	os.Exit(0)
	return err // Unreachable but required for function signature.
}

// loadConfig loads values from flags; if not present, falls back to environment variables.
func (cmd *DaemonCmd) loadConfig() {
	if strings.TrimSpace(cmd.Timeout) == "" {
		cmd.Timeout = os.Getenv(devcontainer.TimeoutExtraEnvVar)
	}
	if strings.TrimSpace(cmd.AccessKey) == "" {
		cmd.AccessKey = os.Getenv(devcontainer.AccessKeyExtraEnvVar)
	}
	if strings.TrimSpace(cmd.PlatformHost) == "" {
		cmd.PlatformHost = os.Getenv(devcontainer.PlatformHostExtraEnvVar)
	}
	if strings.TrimSpace(cmd.WorkspaceHost) == "" {
		cmd.WorkspaceHost = os.Getenv(devcontainer.WorkspaceHostExtraEnvVar)
	}
}

// shouldRunNetworkServer returns true if all necessary network parameters are set.
func (cmd *DaemonCmd) shouldRunNetworkServer() bool {
	return cmd.AccessKey != "" && cmd.PlatformHost != "" && cmd.WorkspaceHost != ""
}

// setupActivityFile creates and sets permissions on the container activity file.
func setupActivityFile() error {
	if err := os.WriteFile(agent.ContainerActivityFile, nil, 0777); err != nil {
		return err
	}
	return os.Chmod(agent.ContainerActivityFile, 0777)
}

// runTimeoutMonitor monitors the activity file and sends an error if stale.
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
			if stat.ModTime().Add(duration).After(time.Now()) {
				continue
			}
			errChan <- errors.New("timeout reached, terminating daemon")
			return
		}
	}
}

// runNetworkServer starts the network server.
func runNetworkServer(ctx context.Context, cmd *DaemonCmd, errChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()
	rootDir := filepath.Join(os.TempDir(), "devpod")
	if err := os.MkdirAll(rootDir, os.ModePerm); err != nil {
		errChan <- err
		return
	}
	logger := initLogging(rootDir)
	config := client.NewConfig()
	config.AccessKey = cmd.AccessKey
	config.Host = "https://" + cmd.PlatformHost
	config.Insecure = true
	baseClient := client.NewClientFromConfig(config)
	if err := baseClient.RefreshSelf(ctx); err != nil {
		errChan <- fmt.Errorf("failed to refresh client: %w", err)
		return
	}
	tsServer := ts.NewWorkspaceServer(&ts.WorkspaceServerConfig{
		AccessKey:     cmd.AccessKey,
		PlatformHost:  ts.RemoveProtocol(cmd.PlatformHost),
		WorkspaceHost: cmd.WorkspaceHost,
		Client:        baseClient,
		RootDir:       rootDir,
	}, logger)
	if err := tsServer.Start(ctx); err != nil {
		errChan <- fmt.Errorf("failed to start network server: %w", err)
	}
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
func initLogging(rootDir string) log.Logger {
	logLevel := logrus.InfoLevel
	logPath := filepath.Join(rootDir, "daemon.log")
	logger := log.NewFileLogger(logPath, logLevel)
	if os.Getenv("DEVPOD_UI") != "true" {
		logger = devpodlog.NewCombinedLogger(logLevel, logger, log.NewStreamLogger(os.Stdout, os.Stderr, logLevel))
	}
	return logger
}
