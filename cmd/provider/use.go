package provider

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	options2 "github.com/loft-sh/devpod/pkg/options"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// UseCmd holds the use cmd flags
type UseCmd struct {
	flags.GlobalFlags

	Reconfigure bool
	Single      bool
	Options     []string
}

// NewUseCmd creates a new destroy command
func NewUseCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &UseCmd{
		GlobalFlags: *flags,
	}
	useCmd := &cobra.Command{
		Use:   "use",
		Short: "Configure an existing provider and set as default",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("please specify the provider to use")
			}

			return cmd.Run(context.Background(), args[0])
		},
	}

	AddFlags(useCmd, cmd)
	return useCmd
}

func AddFlags(useCmd *cobra.Command, cmd *UseCmd) {
	useCmd.Flags().BoolVar(&cmd.Reconfigure, "reconfigure", false, "If enabled will not merge existing provider config")
	useCmd.Flags().BoolVar(&cmd.Single, "single", false, "If enabled DevPod will create a single server for all workspaces")
	useCmd.Flags().StringSliceVarP(&cmd.Options, "option", "o", []string{}, "Provider option in the form KEY=VALUE")
}

// Run runs the command logic
func (cmd *UseCmd) Run(ctx context.Context, providerName string) error {
	if cmd.Context != "" {
		return fmt.Errorf("cannot use --context for this command")
	}

	devPodConfig, err := config.LoadConfig(cmd.Context)
	if err != nil {
		return err
	}

	providerWithOptions, err := workspace.FindProvider(devPodConfig, providerName, log.Default)
	if err != nil {
		return err
	}

	// should reconfigure?
	shouldReconfigure := cmd.Reconfigure || len(cmd.Options) > 0 || !providerWithOptions.Configured
	if shouldReconfigure {
		// parse options
		options, err := provider2.ParseOptions(providerWithOptions.Config, cmd.Options)
		if err != nil {
			return errors.Wrap(err, "parse options")
		}

		// merge with old values
		if !cmd.Reconfigure {
			for k, v := range providerWithOptions.Options {
				_, ok := options[k]
				if !ok && v.UserProvided {
					options[k] = v.Value
				}
			}
		}

		stdout := log.Default.Writer(logrus.InfoLevel, false)
		defer stdout.Close()

		stderr := log.Default.Writer(logrus.ErrorLevel, false)
		defer stderr.Close()

		// run init command
		err = clientimplementation.RunCommandWithBinaries(
			ctx,
			"init",
			providerWithOptions.Config.Exec.Init,
			devPodConfig.DefaultContext,
			nil,
			nil,
			nil,
			providerWithOptions.Config,
			nil,
			nil,
			stdout,
			stderr,
			log.Default,
		)
		if err != nil {
			return errors.Wrap(err, "init")
		}

		// fill defaults
		devPodConfig, err = options2.ResolveOptions(ctx, devPodConfig, providerWithOptions.Config, options, log.Default)
		if err != nil {
			return errors.Wrap(err, "resolve options")
		}

		// run init command
		err = clientimplementation.RunCommandWithBinaries(
			ctx,
			"validate",
			providerWithOptions.Config.Exec.Validate,
			devPodConfig.DefaultContext,
			nil,
			nil,
			devPodConfig.ProviderOptions(providerWithOptions.Config.Name),
			providerWithOptions.Config,
			nil,
			nil,
			stdout,
			stderr,
			log.Default,
		)
		if err != nil {
			return errors.Wrap(err, "validate")
		}
	} else {
		log.Default.Infof("To reconfigure provider %s, run with '--reconfigure' to reconfigure the provider", providerWithOptions.Config.Name)
	}

	// set options
	defaultContext := devPodConfig.Current()
	defaultContext.DefaultProvider = providerWithOptions.Config.Name

	// save provider config
	err = config.SaveConfig(devPodConfig)
	if err != nil {
		return errors.Wrap(err, "save config")
	}

	// print success message
	if shouldReconfigure {
		log.Default.Donef("Successfully configured provider '%s'", providerWithOptions.Config.Name)
	} else {
		log.Default.Donef("Successfully switched default provider to '%s'", providerWithOptions.Config.Name)
	}

	return nil
}
