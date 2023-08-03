package engine

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/devpod/cmd/flags"
	providercmd "github.com/loft-sh/devpod/cmd/provider"
	"github.com/loft-sh/devpod/pkg/config"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DeleteCmd holds the delete cmd flags
type DeleteCmd struct {
	*flags.GlobalFlags

	IgnoreNotFound bool
}

// NewDeleteCmd creates a new command
func NewDeleteCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: flags,
	}
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete or logout from a Loft DevPod engine",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	deleteCmd.Flags().BoolVar(&cmd.IgnoreNotFound, "ignore-not-found", false, "Treat \"engine not found\" as a successful delete")
	return deleteCmd
}

func (cmd *DeleteCmd) Run(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("please specify an engine to delete")
	}

	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	// load engine config
	engineName := args[0]
	engineConfig, err := provider2.LoadEngineConfig(devPodConfig.DefaultContext, engineName)
	if err != nil {
		if os.IsNotExist(err) && cmd.IgnoreNotFound {
			return nil
		}

		return fmt.Errorf("load engine %s: %w", engineName, err)
	}

	// delete the provider
	err = providercmd.DeleteProvider(devPodConfig, engineConfig.ID, true)
	if err != nil {
		return err
	}

	// delete the engine dir itself
	engineDir, err := provider2.GetEngineDir(devPodConfig.DefaultContext, engineConfig.ID)
	if err != nil {
		return err
	}

	// remove engine dir
	err = os.RemoveAll(engineDir)
	if err != nil {
		return errors.Wrap(err, "delete engine dir")
	}

	log.Default.Donef("Successfully deleted engine '%s'", engineName)
	return nil
}
