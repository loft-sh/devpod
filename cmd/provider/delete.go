package provider

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
)

// DeleteCmd holds the delete cmd flags
type DeleteCmd struct {
	*flags.GlobalFlags
}

// NewDeleteCmd creates a new command
func NewDeleteCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: flags,
	}
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a provider",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	return deleteCmd
}

func (cmd *DeleteCmd) Run(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("please specify a provider to delete")
	}

	devPodConfig, err := config.LoadConfig(cmd.Context)
	if err != nil {
		return err
	}

	providerDir, err := provider2.GetProviderDir(devPodConfig.DefaultContext, args[0])
	if err != nil {
		return err
	}

	_, err = os.Stat(providerDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("provider '%s' does not exist", args[0])
		}

		return err
	}

	if devPodConfig.Current().DefaultProvider == args[0] {
		devPodConfig.Current().DefaultProvider = ""
	}
	delete(devPodConfig.Current().Providers, args[0])
	err = config.SaveConfig(devPodConfig)
	if err != nil {
		return errors.Wrap(err, "save config")
	}

	err = os.RemoveAll(providerDir)
	if err != nil {
		return errors.Wrap(err, "delete provider dir")
	}

	log.Default.Donef("Successfully deleted provider '%s'", args[0])
	return nil
}
