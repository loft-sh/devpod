package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DeleteCmd holds the delete cmd flags
type DeleteCmd struct {
	flags.GlobalFlags
}

// NewDeleteCmd creates a new command
func NewDeleteCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: *flags,
	}
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes an existing workspace",
		RunE: func(_ *cobra.Command, args []string) error {
			devPodConfig, err := config.LoadConfig(cmd.Context)
			if err != nil {
				return err
			}

			workspace, provider, err := workspace2.GetWorkspace(devPodConfig, args, log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), workspace, provider)
		},
	}

	return deleteCmd
}

// Run runs the command logic
func (cmd *DeleteCmd) Run(ctx context.Context, workspace *provider2.Workspace, provider provider2.Provider) error {
	workspaceProvider, ok := provider.(provider2.WorkspaceProvider)
	if ok {
		err := cmd.destroyWorkspace(ctx, workspace, workspaceProvider)
		if err != nil {
			return errors.Wrap(err, "destroy workspace")
		}
	}

	serverProvider, ok := provider.(provider2.ServerProvider)
	if ok {
		err := cmd.destroyServer(ctx, workspace, serverProvider)
		if err != nil {
			return errors.Wrap(err, "destroy server")
		}
	}

	return nil
}

func (cmd *DeleteCmd) destroyWorkspace(ctx context.Context, workspace *provider2.Workspace, provider provider2.WorkspaceProvider) error {
	// get instance status
	instanceStatus, err := provider.Status(ctx, workspace, provider2.WorkspaceStatusOptions{})
	if err != nil {
		return err
	} else if instanceStatus == provider2.StatusNotFound {
		return fmt.Errorf("cannot destroy workspace because it couldn't be found")
	}

	// destroy environment
	err = provider.Delete(ctx, workspace, provider2.WorkspaceDeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (cmd *DeleteCmd) destroyServer(ctx context.Context, workspace *provider2.Workspace, provider provider2.ServerProvider) error {
	// get instance status
	instanceStatus, err := provider.Status(ctx, workspace, provider2.StatusOptions{})
	if err != nil {
		return err
	} else if instanceStatus == provider2.StatusNotFound {
		return fmt.Errorf("cannot destroy instance because it couldn't be found")
	}

	// destroy environment
	err = provider.Delete(ctx, workspace, provider2.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}
