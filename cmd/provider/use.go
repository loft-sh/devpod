package provider

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/binaries"
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

	devPodConfig, err := config.LoadConfig("")
	if err != nil {
		return err
	}

	providerWithOptions, err := workspace.FindProvider(devPodConfig, providerName, log.Default)
	if err != nil {
		return err
	}

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
	err = clientimplementation.RunCommand(ctx, providerWithOptions.Config.Exec.Init, provider2.ToEnvironment(nil, nil, nil, nil), nil, stdout, stderr)
	if err != nil {
		return errors.Wrap(err, "init")
	}

	// fill defaults
	devPodConfig, err = options2.ResolveOptions(ctx, devPodConfig, providerWithOptions.Config, options, log.Default)
	if err != nil {
		return errors.Wrap(err, "resolve options")
	}

	// download provider binaries
	if len(providerWithOptions.Config.Binaries) > 0 {
		binariesDir, err := provider2.GetProviderBinariesDir(devPodConfig.DefaultContext, providerWithOptions.Config.Name)
		if err != nil {
			return err
		}

		_, err = binaries.DownloadBinaries(providerWithOptions.Config.Binaries, binariesDir, log.Default)
		if err != nil {
			return errors.Wrap(err, "download binaries")
		}
	}

	// run validate command
	err = clientimplementation.RunCommand(ctx, providerWithOptions.Config.Exec.Validate, provider2.ToEnvironment(nil, nil, devPodConfig.Current().Providers[providerWithOptions.Config.Name].Options, nil), nil, stdout, stderr)
	if err != nil {
		return errors.Wrap(err, "validate")
	}

	// set options
	defaultContext := devPodConfig.Current()
	defaultContext.DefaultProvider = providerWithOptions.Config.Name

	// save provider config
	err = config.SaveConfig(devPodConfig)
	if err != nil {
		return errors.Wrap(err, "save config")
	}

	log.Default.Donef("Successfully configured provider %s, run with '--reconfigure' to reconfigure the provider", providerWithOptions.Config.Name)
	return nil
}
