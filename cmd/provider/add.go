package provider

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// AddCmd holds the cmd flags
type AddCmd struct {
	*flags.GlobalFlags

	Use           bool
	SingleMachine bool
	Options       []string

	Name string
}

// NewAddCmd creates a new command
func NewAddCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &AddCmd{
		GlobalFlags: flags,
	}
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Adds a new provider to DevPod",
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()
			devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, devPodConfig, args)
		},
	}

	addCmd.Flags().BoolVar(&cmd.SingleMachine, "single-machine", false, "If enabled will use a single machine for all workspaces")
	addCmd.Flags().StringVar(&cmd.Name, "name", "", "The name to use for this provider. If empty will use the name within the loaded config")
	addCmd.Flags().BoolVar(&cmd.Use, "use", true, "If enabled will automatically activate the provider")
	addCmd.Flags().StringSliceVarP(&cmd.Options, "option", "o", []string{}, "Provider option in the form KEY=VALUE")
	return addCmd
}

func (cmd *AddCmd) Run(ctx context.Context, devPodConfig *config.Config, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("please specify either a local file, url or git repository. E.g. devpod provider add https://path/to/my/provider.yaml")
	} else if cmd.Name != "" && provider.ProviderNameRegEx.MatchString(cmd.Name) {
		return fmt.Errorf("provider name can only include smaller case letters, numbers or dashes")
	} else if cmd.Name != "" && len(cmd.Name) > 32 {
		return fmt.Errorf("provider name cannot be longer than 32 characters")
	}

	providerConfig, err := workspace.AddProvider(devPodConfig, cmd.Name, args[0], log.Default)
	if err != nil {
		return err
	}

	log.Default.Donef("Successfully installed provider %s", providerConfig.Name)
	if cmd.Use {
		configureErr := configureProvider(ctx, providerConfig, devPodConfig.DefaultContext, cmd.Options, true, &cmd.SingleMachine)
		if configureErr != nil {
			devPodConfig, err = config.LoadConfig(cmd.Context, "")
			if err != nil {
				return err
			}

			err = deleteProvider(devPodConfig, providerConfig.Name)
			if err != nil {
				return errors.Wrap(err, "delete provider")
			}

			return errors.Wrap(configureErr, "configure provider")
		}

		return nil
	}

	log.Default.Infof("To use the provider, please run the following command:")
	log.Default.Infof("devpod provider use %s", providerConfig.Name)
	return nil
}
