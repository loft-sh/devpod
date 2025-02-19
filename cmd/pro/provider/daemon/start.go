package daemon

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/platform"
	platformdaemon "github.com/loft-sh/devpod/pkg/platform/daemon"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// StartCmd holds the cmd flags
type StartCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

// NewStartCmd creates a new command
func NewStartCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &StartCmd{
		GlobalFlags: globalFlags,
	}
	c := &cobra.Command{
		Use:    "start",
		Short:  "Start the daemon",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return c
}

func (cmd *StartCmd) Run(ctx context.Context) error {
	daemonDir := os.Getenv(platform.DaemonFolderEnv)
	if daemonDir == "" {
		return fmt.Errorf("no folder env var for daemon")
	}

	d, err := platformdaemon.Init(ctx, daemonDir, cmd.Debug)
	if err != nil {
		return fmt.Errorf("init daemon: %w", err)
	}

	// NOTE: Do not remove, other processes rely on this for the startup sequence
	logInitialized()

	return d.Start(ctx)
}

func logInitialized() {
	logger := log.NewStreamLogger(os.Stdout, os.Stderr, logrus.InfoLevel)
	logger.SetFormat(log.JSONFormat)
	logger.Done("Initilized daemon")
}
