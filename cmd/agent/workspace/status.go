package workspace

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// StatusCmd holds the cmd flags
type StatusCmd struct {
	*flags.GlobalFlags

	WorkspaceInfo string
}

// NewStatusCmd creates a new command
func NewStatusCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &StatusCmd{
		GlobalFlags: flags,
	}
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Print the status of a remote container",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background(), log.Default.ErrorStreamOnly())
		},
	}
	statusCmd.Flags().StringVar(&cmd.WorkspaceInfo, "workspace-info", "", "The workspace info")
	_ = statusCmd.MarkFlagRequired("workspace-info")
	return statusCmd
}

func (cmd *StatusCmd) Run(ctx context.Context, log log.Logger) error {
	// get workspace
	shouldExit, workspaceInfo, err := agent.WorkspaceInfo(cmd.WorkspaceInfo, log)
	if err != nil {
		return err
	} else if shouldExit {
		return nil
	}

	// create runner
	runner, err := CreateRunner(workspaceInfo, log)
	if err != nil {
		return err
	}

	// find dev container
	containerDetails, err := runner.Find(ctx)
	if err != nil {
		return err
	} else if containerDetails == nil {
		fmt.Print(client.StatusNotFound)
		return nil
	}

	// is running?
	if strings.ToLower(containerDetails.State.Status) == "running" {
		fmt.Print(client.StatusRunning)
		return nil
	} else if strings.ToLower(containerDetails.State.Status) == "exited" {
		fmt.Print(client.StatusStopped)
		return nil
	}

	fmt.Print(client.StatusBusy)
	return nil
}
