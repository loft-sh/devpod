package cmd

import (
	"context"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/spf13/cobra"
)

// CreateCmd holds the create cmd flags
type CreateCmd struct{}

// NewCreateCmd creates a new create command
func NewCreateCmd() *cobra.Command {
	cmd := &CreateCmd{}
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new container",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), provider.FromEnvironment())
		},
	}

	return createCmd
}

// Run runs the command logic
func (cmd *CreateCmd) Run(ctx context.Context, workspace *provider.Workspace) error {
	err := NewDockerProvider().newRunner(workspace, log.Default).Up()
	if err != nil {
		return err
	}

	return nil
}
