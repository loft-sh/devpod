package workspace

import (
	"context"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
)

// LogsDaemonCmd holds the cmd flags
type LogsDaemonCmd struct {
	*flags.GlobalFlags

	ID string
}

// NewLogsDaemonCmd creates a new command
func NewLogsDaemonCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &LogsDaemonCmd{
		GlobalFlags: flags,
	}
	logsDaemonCmd := &cobra.Command{
		Use:   "logs-daemon",
		Short: "Returns the daemon logs",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	logsDaemonCmd.Flags().StringVar(&cmd.ID, "id", "", "The workspace id")
	_ = logsDaemonCmd.MarkFlagRequired("id")
	return logsDaemonCmd
}

func (cmd *LogsDaemonCmd) Run(ctx context.Context) error {
	// get workspace
	shouldExit, _, err := agent.ReadAgentWorkspaceInfo(cmd.AgentDir, cmd.Context, cmd.ID, log.Default.ErrorStreamOnly())
	if err != nil {
		return err
	} else if shouldExit {
		return nil
	}

	logFolder, err := agent.GetAgentDaemonLogFolder(cmd.AgentDir)
	if err != nil {
		return err
	}

	f, err := os.Open(filepath.Join(logFolder, "agent-daemon.log"))
	if err != nil {
		return errors.Wrap(err, "open agent-daemon.log")
	}
	defer f.Close()

	_, err = io.Copy(os.Stdout, f)
	return err
}
