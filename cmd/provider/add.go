package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// AddCmd holds the cmd flags
type AddCmd struct {
	*flags.GlobalFlags

	Use           bool
	SingleMachine bool
	Options       []string

	Name         string
	FromExisting string
}

// NewAddCmd creates a new command
func NewAddCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &AddCmd{
		GlobalFlags: flags,
	}
	addCmd := &cobra.Command{
		Use:   "add [URL or path]",
		Short: "Adds a new provider to DevPod",
		PreRunE: func(cobraCommand *cobra.Command, args []string) error {
			if cmd.FromExisting != "" {
				return cobraCommand.MarkFlagRequired("name")
			}

			return nil
		},
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
	addCmd.Flags().StringVar(&cmd.FromExisting, "from-existing", "", "The name of an existing provider to use as a template. Needs to be used in conjunction with the --name flag")
	addCmd.Flags().BoolVar(&cmd.Use, "use", true, "If enabled will automatically activate the provider")
	addCmd.Flags().StringArrayVarP(&cmd.Options, "option", "o", []string{}, "Provider option in the form KEY=VALUE")

	return addCmd
}

func (cmd *AddCmd) Run(ctx context.Context, devPodConfig *config.Config, args []string) error {
	if len(args) != 1 && cmd.FromExisting == "" {
		return fmt.Errorf("please specify either a local file, url or git repository. E.g. devpod provider add https://path/to/my/provider.yaml")
	} else if cmd.Name != "" && provider.ProviderNameRegEx.MatchString(cmd.Name) {
		return fmt.Errorf("provider name can only include smaller case letters, numbers or dashes")
	} else if cmd.Name != "" && len(cmd.Name) > 32 {
		return fmt.Errorf("provider name cannot be longer than 32 characters")
	} else if cmd.FromExisting != "" && devPodConfig.Current() != nil && devPodConfig.Current().Providers[cmd.FromExisting] == nil {
		return fmt.Errorf("provider %s does not exist", cmd.FromExisting)
	}

	var providerConfig *provider.ProviderConfig
	var options []string
	if cmd.FromExisting != "" {
		providerWithOptions, err := workspace.CloneProvider(devPodConfig, cmd.Name, cmd.FromExisting, log.Default)
		if err != nil {
			return err
		}

		providerConfig = providerWithOptions.Config
		options = mergeOptions(providerWithOptions.Config.Options, providerWithOptions.State.Options, cmd.Options)
	} else {
		c, err := workspace.AddProvider(devPodConfig, cmd.Name, args[0], log.Default)
		if err != nil {
			return err
		}
		providerConfig = c
		options = cmd.Options
	}

	log.Default.Donef("Successfully installed provider %s", providerConfig.Name)
	if cmd.Use {
		configureErr := ConfigureProvider(ctx, providerConfig, devPodConfig.DefaultContext, options, true, false, false, &cmd.SingleMachine, log.Default)
		if configureErr != nil {
			devPodConfig, err := config.LoadConfig(cmd.Context, "")
			if err != nil {
				return err
			}

			err = DeleteProvider(ctx, devPodConfig, providerConfig.Name, true, true, log.Default)
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

// mergeOptions combines user options with existing options, user provided options take precedence
func mergeOptions(desiredOptions map[string]*types.Option, stateOptions map[string]config.OptionValue, userOptions []string) []string {
	retOptions := []string{}
	for key := range desiredOptions {
		userOption, ok := getUserOption(userOptions, key)
		if ok {
			retOptions = append(retOptions, userOption)
			continue
		}
		stateOption, ok := stateOptions[key]
		if !ok {
			continue
		}
		retOptions = append(retOptions, fmt.Sprintf("%s=%s", key, stateOption.Value))
	}

	return retOptions
}

func getUserOption(allOptions []string, optionKey string) (string, bool) {
	if len(allOptions) == 0 {
		return "", false
	}

	for _, option := range allOptions {
		splitted := strings.Split(option, "=")
		if len(splitted) == 1 {
			// ignore
			continue
		}
		if splitted[0] == optionKey {
			return option, true
		}
	}

	return "", false
}
