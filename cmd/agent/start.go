package agent

import (
	"context"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// StartCmd holds the cmd flags
type StartCmd struct {
	*flags.GlobalFlags

	ID string
}

// NewStartCmd creates a new command
func NewStartCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &StartCmd{
		GlobalFlags: flags,
	}
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Starts up a new workspace on the server",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	startCmd.Flags().StringVar(&cmd.ID, "id", "", "The workspace id to start on the agent side")
	_ = startCmd.MarkFlagRequired("id")
	return startCmd
}

func (cmd *StartCmd) Run(ctx context.Context) error {
	// get workspace
	workspaceInfo, err := readAgentWorkspaceInfo(cmd.Context, cmd.ID)
	if err != nil {
		return err
	}

	// check if we need to become root
	shouldExit, err := rerunAsRoot(workspaceInfo)
	if err != nil {
		return errors.Wrap(err, "rerun as root")
	} else if shouldExit {
		return nil
	}

	// start docker container
	err = startContainer(workspaceInfo, log.Default)
	if err != nil {
		return errors.Wrap(err, "start container")
	}

	return nil
}

func startContainer(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error {
	log.Debugf("Starting DevPod container...")
	_, err := createRunner(workspaceInfo, log).Up()
	if err != nil {
		return err
	}
	log.Debugf("Successfully started DevPod container")

	return nil
}
