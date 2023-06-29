package workspace

import (
	"context"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// StopCmd holds the cmd flags
type StopCmd struct {
	*flags.GlobalFlags

	ID string
}

// NewStopCmd creates a new command
func NewStopCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &StopCmd{
		GlobalFlags: flags,
	}
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stops a workspace on the remote server",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	stopCmd.Flags().StringVar(&cmd.ID, "id", "", "The workspace id to stop on the agent side")
	_ = stopCmd.MarkFlagRequired("id")
	return stopCmd
}

func (cmd *StopCmd) Run(ctx context.Context) error {
	// get workspace
	shouldExit, workspaceInfo, err := agent.ReadAgentWorkspaceInfo(cmd.AgentDir, cmd.Context, cmd.ID, log.Default)
	if err != nil {
		return err
	} else if shouldExit {
		return nil
	}

	// stop docker container
	err = stopContainer(ctx, workspaceInfo, log.Default)
	if err != nil {
		return errors.Wrap(err, "stop container")
	}

	return nil
}

func stopContainer(ctx context.Context, workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error {
	log.Debugf("Stopping DevPod container...")
	runner, err := CreateRunner(workspaceInfo, log)
	if err != nil {
		return err
	}

	err = runner.Stop(ctx)
	if err != nil {
		return err
	}
	log.Debugf("Successfully stopped DevPod container")

	return nil
}
