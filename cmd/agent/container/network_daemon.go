package container

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/loft-sh/devpod/cmd/flags"
	devpodlog "github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NetworkDaemonCmd holds the cmd flags
type NetworkDaemonCmd struct {
	*flags.GlobalFlags

	AccessKey     string
	PlatformHost  string
	WorkspaceHost string
}

// NewDaemonCmd creates a new command
func NewNetworkDaemonCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &NetworkDaemonCmd{
		GlobalFlags: flags,
	}
	daemonCmd := &cobra.Command{
		Use:   "network-daemon",
		Short: "Starts tailscale network daemon",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}
	daemonCmd.Flags().StringVar(&cmd.AccessKey, "access-key", "", "")
	daemonCmd.Flags().StringVar(&cmd.PlatformHost, "platform-host", "", "")
	daemonCmd.Flags().StringVar(&cmd.WorkspaceHost, "workspace-host", "", "")
	return daemonCmd
}

// Run runs the command logic
func (cmd *NetworkDaemonCmd) Run(ctx context.Context) error {
	rootDir := filepath.Join(os.TempDir(), "devpod")
	err := os.MkdirAll(rootDir, os.ModePerm)
	if err != nil {
		return err
	}
	log := initLogging(rootDir)

	// init kube config
	config := client.NewConfig()
	config.AccessKey = cmd.AccessKey
	config.Host = "https://" + cmd.PlatformHost
	config.Insecure = true
	baseClient := client.NewClientFromConfig(config)
	err = baseClient.RefreshSelf(ctx)
	if err != nil {
		return err
	}

	tsServer := ts.NewWorkspaceServer(&ts.WorkspaceServerConfig{
		AccessKey: cmd.AccessKey,
		Host:      ts.RemoveProtocol(cmd.PlatformHost),
		Hostname:  cmd.WorkspaceHost,
		Client:    baseClient,
		RootDir:   rootDir,
	}, log)
	err = tsServer.Start(ctx)
	if err != nil {
		return fmt.Errorf("cannot start tsNet server: %w", err)
	}

	return nil
}

func initLogging(rootDir string) log.Logger {
	logLevel := logrus.InfoLevel
	logPath := filepath.Join(rootDir, "daemon.log")
	logger := log.NewFileLogger(logPath, logLevel)
	if os.Getenv("DEVPOD_UI") != "true" {
		logger = devpodlog.NewCombinedLogger(logLevel, logger, log.NewStreamLogger(os.Stdout, os.Stderr, logLevel))
	}

	return logger
}
