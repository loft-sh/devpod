package provider

import (
	"context"
	"fmt"
	"io"

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

	Reconfigure   bool
	SingleMachine bool
	Options       []string
}

// NewUseCmd creates a new command
func NewUseCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &UseCmd{
		GlobalFlags: *flags,
	}
	useCmd := &cobra.Command{
		Use:   "use",
		Short: "Configure an existing provider and set as default",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("please specify the provider to use")
			}

			return cmd.Run(context.Background(), args[0])
		},
	}

	AddFlags(useCmd, cmd)
	return useCmd
}

func AddFlags(useCmd *cobra.Command, cmd *UseCmd) {
	useCmd.Flags().BoolVar(&cmd.SingleMachine, "single-machine", false, "If enabled will use a single machine for all workspaces")
	useCmd.Flags().BoolVar(&cmd.Reconfigure, "reconfigure", false, "If enabled will not merge existing provider config")
	useCmd.Flags().StringSliceVarP(&cmd.Options, "option", "o", []string{}, "Provider option in the form KEY=VALUE")
}

// Run runs the command logic
func (cmd *UseCmd) Run(ctx context.Context, providerName string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	providerWithOptions, err := workspace.FindProvider(devPodConfig, providerName, log.Default)
	if err != nil {
		return err
	}

	// should reconfigure?
	shouldReconfigure := cmd.Reconfigure || len(cmd.Options) > 0 || providerWithOptions.State == nil || cmd.SingleMachine
	if shouldReconfigure {
		return configureProvider(ctx, providerWithOptions.Config, devPodConfig.DefaultContext, cmd.Options, cmd.Reconfigure, &cmd.SingleMachine)
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
	log.Default.Donef("Successfully switched default provider to '%s'", providerWithOptions.Config.Name)
	return nil
}

func configureProvider(ctx context.Context, provider *provider2.ProviderConfig, context string, userOptions []string, reconfigure bool, singleMachine *bool) error {
	// set options
	devPodConfig, err := setOptions(ctx, provider, context, userOptions, reconfigure, false, singleMachine)
	if err != nil {
		return err
	}

	// set options
	defaultContext := devPodConfig.Current()
	defaultContext.DefaultProvider = provider.Name

	// save provider config
	err = config.SaveConfig(devPodConfig)
	if err != nil {
		return errors.Wrap(err, "save config")
	}

	log.Default.Donef("Successfully configured provider '%s'", provider.Name)
	return nil
}

func setOptions(ctx context.Context, provider *provider2.ProviderConfig, context string, userOptions []string, reconfigure, skipRequired bool, singleMachine *bool) (*config.Config, error) {
	devPodConfig, err := config.LoadConfig(context, "")
	if err != nil {
		return nil, err
	}

	// parse options
	options, err := provider2.ParseOptions(provider, userOptions)
	if err != nil {
		return nil, errors.Wrap(err, "parse options")
	}

	// merge with old values
	if !reconfigure {
		for k, v := range devPodConfig.ProviderOptions(provider.Name) {
			_, ok := options[k]
			if !ok && v.UserProvided {
				options[k] = v.Value
			}
		}
	}

	// fill defaults
	devPodConfig, err = options2.ResolveOptions(ctx, devPodConfig, provider, options, skipRequired, singleMachine, log.Default)
	if err != nil {
		return nil, errors.Wrap(err, "resolve options")
	}

	// run init command
	stdout := log.Default.Writer(logrus.InfoLevel, false)
	defer stdout.Close()

	stderr := log.Default.Writer(logrus.ErrorLevel, false)
	defer stderr.Close()

	err = initProvider(ctx, devPodConfig, provider, stdout, stderr)
	if err != nil {
		return nil, err
	}

	return devPodConfig, nil
}

func initProvider(ctx context.Context, devPodConfig *config.Config, provider *provider2.ProviderConfig, stdout, stderr io.Writer) error {
	// run init command
	err := clientimplementation.RunCommandWithBinaries(
		ctx,
		"init",
		provider.Exec.Init,
		devPodConfig.DefaultContext,
		nil,
		nil,
		devPodConfig.ProviderOptions(provider.Name),
		provider,
		nil,
		nil,
		stdout,
		stderr,
		log.Default,
	)
	if err != nil {
		return errors.Wrap(err, "init")
	}
	if devPodConfig.Current().Providers == nil {
		devPodConfig.Current().Providers = map[string]*config.ProviderConfig{}
	}
	if devPodConfig.Current().Providers[provider.Name] == nil {
		devPodConfig.Current().Providers[provider.Name] = &config.ProviderConfig{}
	}
	devPodConfig.Current().Providers[provider.Name].Initialized = true
	return nil
}
