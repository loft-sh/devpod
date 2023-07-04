package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/devpod/cmd/flags"
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
		Short: "Delete a provider",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	deleteCmd.Flags().BoolVar(&cmd.IgnoreNotFound, "ignore-not-found", false, "Treat \"provider not found\" as a successful delete")
	return deleteCmd
}

func (cmd *DeleteCmd) Run(ctx context.Context, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("please specify a provider to delete")
	}

	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	provider := devPodConfig.Current().DefaultProvider
	if len(args) > 0 {
		provider = args[0]
	} else if provider == "" {
		return fmt.Errorf("please specify a provider to delete")
	}

	err = deleteProvider(devPodConfig, provider, cmd.IgnoreNotFound)
	if err != nil {
		return err
	}

	log.Default.Donef("Successfully deleted provider '%s'", provider)
	return nil
}

func deleteProvider(devPodConfig *config.Config, provider string, ignoreNotFound bool) error {
	if devPodConfig.Current().DefaultProvider == provider {
		devPodConfig.Current().DefaultProvider = ""
	}
	delete(devPodConfig.Current().Providers, provider)
	err := config.SaveConfig(devPodConfig)
	if err != nil {
		return errors.Wrap(err, "save config")
	}

	providerDir, err := provider2.GetProviderDir(devPodConfig.DefaultContext, provider)
	if err != nil {
		return err
	}
	_, err = os.Stat(providerDir)
	if err != nil {
		if os.IsNotExist(err) {
			if ignoreNotFound {
				return nil
			}

			return fmt.Errorf("provider '%s' does not exist", provider)
		}

		return err
	}
	err = os.RemoveAll(providerDir)
	if err != nil {
		return errors.Wrap(err, "delete provider dir")
	}

	return nil
}
