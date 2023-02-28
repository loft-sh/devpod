package cmd

import (
	"context"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// StopCmd holds the cmd flags
type StopCmd struct{}

// NewStopCmd defines a command
func NewStopCmd() *cobra.Command {
	cmd := &StopCmd{}
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop an instance",
		RunE: func(_ *cobra.Command, args []string) error {
			gcloudProvider, err := newProvider(log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), gcloudProvider, provider.FromEnvironment(), log.Default)
		},
	}

	return stopCmd
}

// Run runs the command logic
func (cmd *StopCmd) Run(ctx context.Context, provider *gcloudProvider, workspace *provider.Workspace, log log.Logger) error {
	name := getName(workspace)
	args := []string{
		"compute",
		"instances",
		"stop",
		name,
		"--project=" + provider.Config.Project,
		"--zone=" + provider.Config.Zone,
		"--async",
	}

	_, err := provider.output(ctx, args...)
	if err != nil {
		return errors.Wrapf(err, "stop vm")
	}

	return nil
}
