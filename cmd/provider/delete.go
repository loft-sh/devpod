package provider

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
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

	defaultProviders := devPodConfig.Contexts[devPodConfig.DefaultContext].Providers
	if defaultProviders == nil || defaultProviders[args[0]] == nil {
		return fmt.Errorf("provider %s is not configured", args[0])
	}

	providerDir, err := config.GetProviderDir(devPodConfig.DefaultContext, args[0])
	if err != nil {
		return err
	}

	if devPodConfig.Contexts[devPodConfig.DefaultContext].DefaultProvider == args[0] {
		devPodConfig.Contexts[devPodConfig.DefaultContext].DefaultProvider = ""
	}
	delete(defaultProviders, args[0])
	devPodConfig.Contexts[devPodConfig.DefaultContext].Providers = defaultProviders
	err = config.SaveConfig(devPodConfig)
	if err != nil {
		return errors.Wrap(err, "save config")
	}

	_ = os.RemoveAll(providerDir)
	return nil
}
