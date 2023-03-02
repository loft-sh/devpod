package cmd

import (
	"context"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DeleteCmd holds the cmd flags
type DeleteCmd struct{}

// NewDeleteCmd defines a command
func NewDeleteCmd() *cobra.Command {
	cmd := &DeleteCmd{}
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an instance",
		RunE: func(_ *cobra.Command, args []string) error {
			gcloudProvider, err := newProvider(log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), gcloudProvider, provider.FromEnvironment(), log.Default)
		},
	}

	return deleteCmd
}

// Run runs the command logic
func (cmd *DeleteCmd) Run(ctx context.Context, provider *gcloudProvider, machine *provider.Machine, log log.Logger) error {
	name := getName(machine)
	args := []string{
		"compute",
		"instances",
		"delete",
		name,
		"--project=" + provider.Config.Project,
		"--zone=" + provider.Config.Zone,
	}

	_, err := provider.output(ctx, args...)
	if err != nil {
		return errors.Wrapf(err, "destroy vm")
	}
	return nil
}

func getName(machine *provider.Machine) string {
	return "devpod-" + machine.ID
}
