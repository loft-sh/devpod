package workspace

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// LogsCmd holds the cmd flags
type LogsCmd struct {
	*flags.GlobalFlags

	ID string
}

// NewLogsCmd creates a new command
func NewLogsCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &LogsCmd{
		GlobalFlags: flags,
	}
	c := &cobra.Command{
		Use:   "logs",
		Short: "Returns the workspace container logs",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	c.Flags().StringVar(&cmd.ID, "id", "", "The workspace id")
	_ = c.MarkFlagRequired("id")

	return c
}

func (cmd *LogsCmd) Run(ctx context.Context) error {
	// get workspace info
	shouldExit, workspaceInfo, err := agent.ReadAgentWorkspaceInfo(cmd.AgentDir, cmd.Context, cmd.ID, log.Default.ErrorStreamOnly())
	if err != nil {
		return err
	} else if shouldExit {
		return nil
	}
	logger := log.Default.ErrorStreamOnly()

	// create new runner
	runner, err := devcontainer.NewRunner(agent.DevPodBinary, agent.DefaultAgentDownloadURL(), workspaceInfo, logger)
	if err != nil {
		return fmt.Errorf("create runner: %w", err)
	}

	// write devcontainer logs to stdout
	return runner.Logs(ctx, os.Stdout)
}
