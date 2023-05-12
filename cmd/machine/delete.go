package machine

import (
	"context"
	"time"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DeleteCmd holds the configuration
type DeleteCmd struct {
	*flags.GlobalFlags

	GracePeriod string
	Force       bool
}

// NewDeleteCmd creates a new destroy command
func NewDeleteCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: flags,
	}
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes an existing machine",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	return deleteCmd
}

// Run runs the command logic
func (cmd *DeleteCmd) Run(ctx context.Context, args []string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	machineClient, err := workspace.GetMachine(devPodConfig, args, log.Default)
	if err != nil {
		return err
	}

	var duration *time.Duration
	if cmd.GracePeriod != "" {
		gracePeriod, err := time.ParseDuration(cmd.GracePeriod)
		if err != nil {
			return errors.Wrap(err, "parse grace-period")
		}

		duration = &gracePeriod
	}

	err = machineClient.Delete(ctx, client.DeleteOptions{
		Force:       cmd.Force,
		GracePeriod: duration,
	})
	if err != nil {
		return err
	}

	return nil
}
