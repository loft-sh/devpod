package context

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"strings"
)

// CreateCmd holds the create cmd flags
type CreateCmd struct {
	flags.GlobalFlags

	Options []string
}

// NewCreateCmd creates a new command
func NewCreateCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &CreateCmd{
		GlobalFlags: *flags,
	}
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new DevPod context",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("please specify the context to create")
			}

			return cmd.Run(context.Background(), args[0])
		},
	}

	createCmd.Flags().StringSliceVarP(&cmd.Options, "option", "o", []string{}, "context option in the form KEY=VALUE")
	return createCmd
}

// Run runs the command logic
func (cmd *CreateCmd) Run(ctx context.Context, context string) error {
	devPodConfig, err := config.LoadConfig("", cmd.Provider)
	if err != nil {
		return err
	} else if devPodConfig.Contexts[context] != nil {
		return fmt.Errorf("context '%s' already exists", context)
	}

	// verify name
	if provider2.ProviderNameRegEx.MatchString(context) {
		return fmt.Errorf("context name can only include smaller case letters, numbers or dashes")
	} else if len(context) > 48 {
		return fmt.Errorf("context name cannot be longer than 48 characters")
	}
	devPodConfig.Contexts[context] = &config.ContextConfig{}

	// check if there are create options set
	if len(cmd.Options) > 0 {
		err = setOptions(devPodConfig, context, cmd.Options)
		if err != nil {
			return err
		}
	}

	devPodConfig.DefaultContext = context
	err = config.SaveConfig(devPodConfig)
	if err != nil {
		return errors.Wrap(err, "save config")
	}

	return nil
}

func setOptions(devPodConfig *config.Config, context string, options []string) error {
	optionValues, err := parseOptions(options)
	if err != nil {
		return err
	} else if devPodConfig.Contexts[context] == nil {
		return fmt.Errorf("context '%s' doesn't exist", context)
	}

	newValues := map[string]config.OptionValue{}
	if devPodConfig.Contexts[context].Options != nil {
		for k, v := range devPodConfig.Contexts[context].Options {
			newValues[k] = v
		}
	}
	for k, v := range optionValues {
		newValues[k] = v
	}

	devPodConfig.Contexts[context].Options = newValues
	return nil
}

func parseOptions(options []string) (map[string]config.OptionValue, error) {
	allowedOptions := []string{}
	contextOptions := map[string]config.ContextOption{}
	for _, option := range config.ContextOptions {
		allowedOptions = append(allowedOptions, option.Name)
		contextOptions[option.Name] = option
	}

	retMap := map[string]config.OptionValue{}
	for _, option := range options {
		splitted := strings.Split(option, "=")
		if len(splitted) == 1 {
			return nil, fmt.Errorf("invalid option '%s', expected format KEY=VALUE", option)
		}

		key := strings.ToUpper(strings.TrimSpace(splitted[0]))
		value := strings.Join(splitted[1:], "=")
		contextOption, ok := contextOptions[key]
		if !ok {
			return nil, fmt.Errorf("invalid option '%s', allowed options are: %v", key, allowedOptions)
		}

		if len(contextOption.Enum) > 0 {
			found := false
			for _, e := range contextOption.Enum {
				if value == e {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("invalid value '%s' for option '%s', has to match one of the following values: %v", value, key, contextOption.Enum)
			}
		}

		retMap[key] = config.OptionValue{
			Value:        value,
			UserProvided: true,
		}
	}

	return retMap, nil
}
