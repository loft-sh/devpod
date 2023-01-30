package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider/types"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// StopCmd holds the destroy cmd flags
type StopCmd struct{}

// NewStopCmd creates a new destroy command
func NewStopCmd() *cobra.Command {
	cmd := &StopCmd{}
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stops an existing workspace",
		RunE: func(_ *cobra.Command, args []string) error {
			workspace, provider, err := workspace2.GetWorkspace(args, log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), workspace, provider)
		},
	}

	return stopCmd
}

// Run runs the command logic
func (cmd *StopCmd) Run(ctx context.Context, workspace *config.Workspace, provider types.Provider) error {
	workspaceProvider, ok := provider.(types.WorkspaceProvider)
	if ok {
		err := cmd.stopWorkspace(ctx, workspace, workspaceProvider)
		if err != nil {
			return errors.Wrap(err, "stop workspace")
		}
	}

	serverProvider, ok := provider.(types.ServerProvider)
	if ok {
		err := cmd.stopServer(ctx, workspace, serverProvider)
		if err != nil {
			return errors.Wrap(err, "stop server")
		}
	}

	return nil
}

func (cmd *StopCmd) stopServer(ctx context.Context, workspace *config.Workspace, provider types.ServerProvider) error {
	// get instance status
	instanceStatus, err := provider.Status(ctx, workspace, types.StatusOptions{})
	if err != nil {
		return err
	} else if instanceStatus != types.StatusRunning {
		return fmt.Errorf("cannot stop instance because it is '%s'", instanceStatus)
	}

	// stop environment
	err = provider.Stop(ctx, workspace, types.StopOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (cmd *StopCmd) stopWorkspace(ctx context.Context, workspace *config.Workspace, provider types.WorkspaceProvider) error {
	// get instance status
	instanceStatus, err := provider.WorkspaceStatus(ctx, workspace, types.WorkspaceStatusOptions{})
	if err != nil {
		return err
	} else if instanceStatus != types.StatusRunning {
		return fmt.Errorf("cannot stop instance because it is '%s'", instanceStatus)
	}

	// stop environment
	err = provider.WorkspaceStop(ctx, workspace, types.WorkspaceStopOptions{})
	if err != nil {
		return err
	}

	return nil
}
