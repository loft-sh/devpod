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

// StopCmd holds the destroy cmd flags
type StopCmd struct {
	*flags.GlobalFlags
}

// NewStopCmd creates a new destroy command
func NewStopCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &StopCmd{
		GlobalFlags: flags,
	}
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stops an existing workspace",
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()
			devPodConfig, err := config.LoadConfig(cmd.Context)
			if err != nil {
				return err
			}

			workspace, provider, err := workspace2.GetWorkspace(ctx, devPodConfig, nil, args, log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, workspace, provider)
		},
	}

	return stopCmd
}

// Run runs the command logic
func (cmd *StopCmd) Run(ctx context.Context, workspace *provider2.Workspace, provider provider2.Provider) error {
	workspaceProvider, ok := provider.(provider2.WorkspaceProvider)
	if ok {
		err := cmd.stopWorkspace(ctx, workspace, workspaceProvider)
		if err != nil {
			return errors.Wrap(err, "stop workspace")
		}
	}

	serverProvider, ok := provider.(provider2.ServerProvider)
	if ok {
		err := cmd.stopServer(ctx, workspace, serverProvider)
		if err != nil {
			return errors.Wrap(err, "stop server")
		}
	}

	return nil
}

func (cmd *StopCmd) stopServer(ctx context.Context, workspace *provider2.Workspace, provider provider2.ServerProvider) error {
	// get instance status
	instanceStatus, err := provider.Status(ctx, workspace, provider2.StatusOptions{})
	if err != nil {
		return err
	} else if instanceStatus != provider2.StatusRunning {
		return fmt.Errorf("cannot stop instance because it is '%s'", instanceStatus)
	}

	// stop environment
	err = provider.Stop(ctx, workspace, provider2.StopOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (cmd *StopCmd) stopWorkspace(ctx context.Context, workspace *provider2.Workspace, provider provider2.WorkspaceProvider) error {
	// get instance status
	instanceStatus, err := provider.Status(ctx, workspace, provider2.WorkspaceStatusOptions{})
	if err != nil {
		return err
	} else if instanceStatus != provider2.StatusRunning {
		return fmt.Errorf("cannot stop instance because it is '%s'", instanceStatus)
	}

	// stop environment
	err = provider.Stop(ctx, workspace, provider2.WorkspaceStopOptions{})
	if err != nil {
		return err
	}

	return nil
}
