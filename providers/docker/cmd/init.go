package cmd

import (
	"context"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/spf13/cobra"
)

// InitCmd holds the cmd flags
type InitCmd struct{}

// NewInitCmd defines a command
func NewInitCmd() *cobra.Command {
	cmd := &InitCmd{}
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Init Docker Provider",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), provider.FromEnvironment())
		},
	}

	return initCmd
}

// Run runs the command logic
func (cmd *InitCmd) Run(ctx context.Context, workspace *provider.Workspace) error {
	return nil
}
