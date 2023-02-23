package workspace

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

// StatusCmd holds the cmd flags
type StatusCmd struct {
	*flags.GlobalFlags

	ID string
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
			return cmd.Run(context.Background())
		},
	}
	statusCmd.Flags().StringVar(&cmd.ID, "id", "", "The workspace id to print the status on the agent side")
	_ = statusCmd.MarkFlagRequired("id")
	return statusCmd
}

func (cmd *StatusCmd) Run(ctx context.Context) error {
	// get workspace folder
	_, err := agent.GetAgentWorkspaceDir(cmd.Context, cmd.ID)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Print(client.StatusNotFound)
			return nil
		}

		return err
	}

	// get workspace
	workspaceInfo, err := agent.ReadAgentWorkspaceInfo(cmd.Context, cmd.ID)
	if err != nil {
		return err
	}

	// check if we need to become root
	shouldExit, err := agent.RerunAsRoot(workspaceInfo)
	if err != nil {
		return errors.Wrap(err, "rerun as root")
	} else if shouldExit {
		return nil
	}

	// find dev container
	containerDetails, err := createRunner(workspaceInfo, log.Default).FindDevContainer()
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
