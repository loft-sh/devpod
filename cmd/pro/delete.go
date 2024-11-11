package pro

import (
	"context"
	"fmt"
	"os"

	proflags "github.com/loft-sh/devpod/cmd/pro/flags"
	providercmd "github.com/loft-sh/devpod/cmd/provider"
	"github.com/loft-sh/devpod/pkg/config"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DeleteCmd holds the delete cmd flags
type DeleteCmd struct {
	*proflags.GlobalFlags

	IgnoreNotFound bool
}

// NewDeleteCmd creates a new command
func NewDeleteCmd(flags *proflags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: flags,
	}
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete or logout from a DevPod Pro Instance",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	deleteCmd.Flags().BoolVar(&cmd.IgnoreNotFound, "ignore-not-found", false, "Treat \"pro instance not found\" as a successful delete")
	return deleteCmd
}

func (cmd *DeleteCmd) Run(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("please specify an pro instance to delete")
	}

	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	// load pro instance config
	proInstanceName := args[0]
	proInstanceConfig, err := provider2.LoadProInstanceConfig(devPodConfig.DefaultContext, proInstanceName)
	if err != nil {
		if os.IsNotExist(err) && cmd.IgnoreNotFound {
			return nil
		}

		return fmt.Errorf("load pro instance %s: %w", proInstanceName, err)
	}

	// delete the provider
	err = providercmd.DeleteProvider(ctx, devPodConfig, proInstanceConfig.Provider, true, true, log.Default)
	if err != nil {
		return err
	}

	// delete the pro instance dir itself
	proInstanceDir, err := provider2.GetProInstanceDir(devPodConfig.DefaultContext, proInstanceConfig.Host)
	if err != nil {
		return err
	}

	err = os.RemoveAll(proInstanceDir)
	if err != nil {
		return errors.Wrap(err, "delete pro instance dir")
	}

	log.Default.Donef("Successfully deleted pro instance '%s'", proInstanceName)
	return nil
}
