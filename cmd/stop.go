package cmd

import (
	"context"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider/gcp"
	"github.com/loft-sh/devpod/pkg/provider/types"
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
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	return stopCmd
}

// Run runs the command logic
func (cmd *StopCmd) Run(ctx context.Context, _ []string) error {
	// TODO: remove hardcode
	provider := gcp.NewGCPProvider(log.Default)
	workspace := &types.Workspace{
		ID:         "test",
		Repository: "https://github.com/microsoft/vscode-course-sample",
	}

	// stop environment
	err := provider.Stop(ctx, workspace, types.StopOptions{})
	if err != nil {
		return err
	}

	return nil
}
