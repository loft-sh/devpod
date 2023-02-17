package provider

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/binaries"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	options2 "github.com/loft-sh/devpod/pkg/provider/options"
	"github.com/loft-sh/devpod/pkg/survey"
	"github.com/loft-sh/devpod/pkg/terminal"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"regexp"
	"strings"
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
	options, err := parseOptions(providerWithOptions.Provider, cmd.Options)
	if err != nil {
		return errors.Wrap(err, "parse options")
	}

	// merge with old values
	if !cmd.Reconfigure {
		for k, v := range providerWithOptions.Options {
			_, ok := options[k]
			if !ok {
				options[k] = v
			}
		}
	}

	// TODO: this is kind of a hack, only to get the options correctly passed to init & validate
	workspaceConfig := &provider2.Workspace{Provider: provider2.WorkspaceProviderConfig{Options: options}}

	// run init command
	err = providerWithOptions.Provider.Init(ctx, workspaceConfig, provider2.InitOptions{})
	if err != nil {
		return err
	}

	// fill defaults
	workspaceConfig.Provider.Options, err = options2.ResolveOptions(ctx, "", "", workspaceConfig, providerWithOptions.Provider)
	if err != nil {
		return errors.Wrap(err, "resolve options")
	}

	// ensure required
	err = ensureRequired(workspaceConfig, providerWithOptions.Provider, log.Default)
	if err != nil {
		return errors.Wrap(err, "ensure required")
	}

	// download provider binaries
	if len(providerWithOptions.Config.Binaries) > 0 {
		binariesDir, err := config.GetProviderBinariesDir(devPodConfig.DefaultContext, providerWithOptions.Config.Name)
		if err != nil {
			return err
		}

		_, err = binaries.DownloadBinaries(providerWithOptions.Config.Binaries, binariesDir, log.Default)
		if err != nil {
			return errors.Wrap(err, "download binaries")
		}
	}

	// run validate command
	err = providerWithOptions.Provider.Validate(ctx, workspaceConfig, provider2.ValidateOptions{})
	if err != nil {
		return err
	}

	// check provider mode
	mode := provider2.ModeMultiple
	if cmd.Single || len(providerWithOptions.Config.Exec.Create) == 0 {
		mode = provider2.ModeSingle
	}

	// set options
	defaultContext := devPodConfig.Contexts[devPodConfig.DefaultContext]
	defaultContext.DefaultProvider = providerWithOptions.Provider.Name()
	defaultContext.Providers[providerName] = &config.ConfigProvider{
		Mode:    mode,
		Options: workspaceConfig.Provider.Options,
	}

	// save provider config
	err = config.SaveConfig(devPodConfig)
	if err != nil {
		return errors.Wrap(err, "save config")
	}

	log.Default.Donef("Successfully configured provider %s, run with '--reconfigure' to reconfigure the provider", providerWithOptions.Config.Name)
	return nil
}

func ensureRequired(workspace *provider2.Workspace, provider provider2.Provider, log log.Logger) error {
	for optionName, option := range provider.Options() {
		if !option.Required {
			continue
		}

		val, ok := workspace.Provider.Options[optionName]
		if !ok || val.Value == "" {
			if !terminal.IsTerminalIn {
				return fmt.Errorf("option %s is required, but no value provided", optionName)
			}

			log.Info(option.Description)
			answer, err := log.Question(&survey.QuestionOptions{
				Question:               fmt.Sprintf("Please enter a value for %s", optionName),
				Options:                option.Enum,
				ValidationRegexPattern: option.ValidationPattern,
				ValidationMessage:      option.ValidationMessage,
			})
			if err != nil {
				return err
			}

			workspace.Provider.Options[optionName] = provider2.OptionValue{
				Value: answer,
				Local: val.Local,
			}
		}
	}

	return nil
}

func parseOptions(provider provider2.Provider, options []string) (map[string]provider2.OptionValue, error) {
	providerOptions := provider.Options()
	if providerOptions == nil {
		providerOptions = map[string]*provider2.ProviderOption{}
	}

	allowedOptions := []string{}
	for optionName := range providerOptions {
		allowedOptions = append(allowedOptions, optionName)
	}

	retMap := map[string]provider2.OptionValue{}
	for _, option := range options {
		splitted := strings.Split(option, "=")
		if len(splitted) == 1 {
			return nil, fmt.Errorf("invalid option %s, expected format KEY=VALUE", option)
		}

		key := strings.ToUpper(strings.TrimSpace(splitted[0]))
		value := strings.Join(splitted[1:], "=")
		providerOption := providerOptions[key]
		if providerOption == nil {
			return nil, fmt.Errorf("invalid option %s, allowed options are: %v", key, allowedOptions)
		}

		if providerOption.ValidationPattern != "" {
			matcher, err := regexp.Compile(providerOption.ValidationPattern)
			if err != nil {
				return nil, err
			}

			if !matcher.MatchString(value) {
				if providerOption.ValidationMessage != "" {
					return nil, fmt.Errorf(providerOption.ValidationMessage)
				}

				return nil, fmt.Errorf("invalid value '%s' for option '%s', has to match the following regEx: %s", value, key, providerOption.ValidationPattern)
			}
		}

		if len(providerOption.Enum) > 0 {
			found := false
			for _, e := range providerOption.Enum {
				if value == e {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("invalid value '%s' for option '%s', has to match one of the following values: %v", value, key, providerOption.Enum)
			}
		}

		retMap[key] = provider2.OptionValue{
			Value: value,
		}
	}

	return retMap, nil
}
