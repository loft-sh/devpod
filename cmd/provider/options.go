package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/log/table"
	"github.com/loft-sh/devpod/pkg/options"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/spf13/cobra"
)

// OptionsCmd holds the options cmd flags
type OptionsCmd struct {
	*flags.GlobalFlags

	Prefill bool
	Hidden  bool
	Output  string
}

// NewOptionsCmd creates a new command
func NewOptionsCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &OptionsCmd{
		GlobalFlags: flags,
	}
	optionsCmd := &cobra.Command{
		Use:   "options",
		Short: "Show options of an existing provider",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	optionsCmd.Flags().BoolVar(&cmd.Prefill, "prefill", true, "If provider is not initialized, will show prefilled values.")
	optionsCmd.Flags().BoolVar(&cmd.Hidden, "hidden", false, "If true, will also show hidden options.")
	optionsCmd.Flags().StringVar(&cmd.Output, "output", "plain", "The output format to use. Can be json or plain")
	return optionsCmd
}

type optionWithValue struct {
	provider2.ProviderOption `json:",inline"`

	Value string `json:"value,omitempty"`
}

// Run runs the command logic
func (cmd *OptionsCmd) Run(ctx context.Context, args []string) error {
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

	provider, err := workspace.FindProvider(devPodConfig, providerName, log.Default.ErrorStreamOnly())
	if err != nil {
		return err
	}

	if cmd.Prefill && (provider.State == nil || !provider.State.Initialized) {
		devPodConfig, err = options.ResolveOptions(ctx, devPodConfig, provider.Config, nil, true, nil, log.Default.ErrorStreamOnly())
		if err != nil {
			return err
		}
	}

	entryOptions := devPodConfig.ProviderOptions(provider.Config.Name)
	if cmd.Output == "plain" {
		tableEntries := [][]string{}
		for optionName, entry := range provider.Config.Options {
			if !cmd.Hidden && entry.Hidden {
				continue
			}

			value := entryOptions[optionName].Value
			if value != "" && entry.Password {
				value = "********"
			}

			tableEntries = append(tableEntries, []string{
				optionName,
				strconv.FormatBool(entry.Required),
				entry.Description,
				entry.Default,
				value,
			})
		}
		sort.SliceStable(tableEntries, func(i, j int) bool {
			return tableEntries[i][0] < tableEntries[j][0]
		})

		table.PrintTable(log.Default, []string{
			"Name",
			"Required",
			"Description",
			"Default",
			"Value",
		}, tableEntries)
	} else if cmd.Output == "json" {
		options := map[string]optionWithValue{}
		for optionName, entry := range provider.Config.Options {
			if !cmd.Hidden && entry.Hidden {
				continue
			}

			options[optionName] = optionWithValue{
				ProviderOption: *entry,
				Value:          entryOptions[optionName].Value,
			}
		}

		out, err := json.MarshalIndent(options, "", "  ")
		if err != nil {
			return err
		}
		fmt.Print(string(out))
	} else {
		return fmt.Errorf("unexpected output format, choose either json or plain. Got %s", cmd.Output)
	}

	return nil
}
