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

// DestroyCmd holds the destroy cmd flags
type DestroyCmd struct {
	Snapshot bool
}

// NewDestroyCmd creates a new destroy command
func NewDestroyCmd() *cobra.Command {
	cmd := &DestroyCmd{}
	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "Destroys an existing workspace",
		RunE: func(_ *cobra.Command, args []string) error {
			workspace, provider, err := workspace2.GetWorkspace(args, log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), workspace, provider)
		},
	}

	destroyCmd.Flags().BoolVar(&cmd.Snapshot, "snapshot", false, "If true will create a snapshot for the environment")
	return destroyCmd
}

// Run runs the command logic
func (cmd *DestroyCmd) Run(ctx context.Context, workspace *config.Workspace, provider types.Provider) error {
	workspaceProvider, ok := provider.(types.WorkspaceProvider)
	if ok {
		err := cmd.destroyWorkspace(ctx, workspace, workspaceProvider)
		if err != nil {
			return errors.Wrap(err, "destroy workspace")
		}
	}

	serverProvider, ok := provider.(types.ServerProvider)
	if ok {
		err := cmd.destroyServer(ctx, workspace, serverProvider)
		if err != nil {
			return errors.Wrap(err, "destroy server")
		}
	}

	return nil
}

func (cmd *DestroyCmd) destroyWorkspace(ctx context.Context, workspace *config.Workspace, provider types.WorkspaceProvider) error {
	// get instance status
	instanceStatus, err := provider.WorkspaceStatus(ctx, workspace, types.WorkspaceStatusOptions{})
	if err != nil {
		return err
	} else if instanceStatus == types.StatusNotFound {
		return fmt.Errorf("cannot destroy workspace because it couldn't be found")
	}

	// destroy environment
	err = provider.WorkspaceDestroy(ctx, workspace, types.WorkspaceDestroyOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (cmd *DestroyCmd) destroyServer(ctx context.Context, workspace *config.Workspace, provider types.ServerProvider) error {
	// get instance status
	instanceStatus, err := provider.Status(ctx, workspace, types.StatusOptions{})
	if err != nil {
		return err
	} else if instanceStatus == types.StatusNotFound {
		return fmt.Errorf("cannot destroy instance because it couldn't be found")
	}

	// destroy environment
	err = provider.Destroy(ctx, workspace, types.DestroyOptions{})
	if err != nil {
		return err
	}

	// destroy snapshot
	if cmd.Snapshot {
		//err = provider.DestroySnapshot(ctx, workspace, types.DestroySnapshotOptions{})
		//if err != nil {
		//	return err
		//}
	}

	return nil
}
