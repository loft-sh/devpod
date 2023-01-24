package cmd

import (
	"context"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider/gcp"
	"github.com/loft-sh/devpod/pkg/provider/types"
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
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	destroyCmd.Flags().BoolVar(&cmd.Snapshot, "snapshot", false, "If true will create a snapshot for the environment")
	return destroyCmd
}

// Run runs the command logic
func (cmd *DestroyCmd) Run(ctx context.Context, _ []string) error {
	// TODO: remove hardcode
	provider := gcp.NewGCPProvider(log.Default)
	workspace := &types.Workspace{
		ID:         "test",
		Repository: "https://github.com/microsoft/vscode-course-sample",
	}

	// destroy environment
	err := provider.Destroy(ctx, workspace, types.DestroyOptions{})
	if err != nil {
		return err
	}

	// destroy snapshot
	if cmd.Snapshot {
		err = provider.DestroySnapshot(ctx, workspace, types.DestroySnapshotOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}
