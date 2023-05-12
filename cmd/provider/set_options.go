package provider

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// SetOptionsCmd holds the use cmd flags
type SetOptionsCmd struct {
	flags.GlobalFlags

	Reconfigure   bool
	SingleMachine bool
	Options       []string
}

// NewSetOptionsCmd creates a new command
func NewSetOptionsCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &SetOptionsCmd{
		GlobalFlags: *flags,
	}
	setOptionsCmd := &cobra.Command{
		Use:   "set-options",
		Short: "Sets options for the given provider. Similar to 'devpod provider use', but does not switch the default provider.",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	setOptionsCmd.Flags().BoolVar(&cmd.SingleMachine, "single-machine", false, "If enabled will use a single machine for all workspaces")
	setOptionsCmd.Flags().BoolVar(&cmd.Reconfigure, "reconfigure", false, "If enabled will not merge existing provider config")
	setOptionsCmd.Flags().StringSliceVarP(&cmd.Options, "option", "o", []string{}, "Provider option in the form KEY=VALUE")
	return setOptionsCmd
}

// Run runs the command logic
func (cmd *SetOptionsCmd) Run(ctx context.Context, args []string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	providerName := devPodConfig.Current().DefaultProvider
	if len(args) > 0 {
		providerName = args[0]
	} else if providerName == "" {
		return fmt.Errorf("please specify a provider")
	}

	providerWithOptions, err := workspace.FindProvider(devPodConfig, providerName, log.Default)
	if err != nil {
		return err
	}

	devPodConfig, err = setOptions(ctx, providerWithOptions.Config, devPodConfig.DefaultContext, cmd.Options, cmd.Reconfigure, false, &cmd.SingleMachine)
	if err != nil {
		return err
	}

	// save provider config
	err = config.SaveConfig(devPodConfig)
	if err != nil {
		return errors.Wrap(err, "save config")
	}

	// print success message
	log.Default.Donef("Successfully set options for provider '%s'", providerWithOptions.Config.Name)
	return nil
}
