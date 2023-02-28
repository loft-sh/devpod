package cmd

import (
	"context"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// StartCmd holds the cmd flags
type StartCmd struct{}

// NewStartCmd defines a command
func NewStartCmd() *cobra.Command {
	cmd := &StartCmd{}
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start an instance",
		RunE: func(_ *cobra.Command, args []string) error {
			gcloudProvider, err := newProvider(log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), gcloudProvider, provider.FromEnvironment(), log.Default)
		},
	}

	return startCmd
}

// Run runs the command logic
func (cmd *StartCmd) Run(ctx context.Context, provider *gcloudProvider, workspace *provider.Workspace, log log.Logger) error {
	name := getName(workspace)
	args := []string{
		"compute",
		"instances",
		"start",
		name,
		"--project=" + provider.Config.Project,
		"--zone=" + provider.Config.Zone,
	}

	_, err := provider.output(ctx, args...)
	if err != nil {
		return errors.Wrapf(err, "start vm")
	}

	return nil
}
