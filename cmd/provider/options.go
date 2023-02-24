package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/log/table"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/spf13/cobra"
	"sort"
	"strconv"
)

// OptionsCmd holds the options cmd flags
type OptionsCmd struct {
	*flags.GlobalFlags

	Hidden bool
	Output string
}

// NewOptionsCmd creates a new destroy command
func NewOptionsCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &OptionsCmd{
		GlobalFlags: flags,
	}
	optionsCmd := &cobra.Command{
		Use:   "options",
		Short: "Show options of an existing provider",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("please specify the provider to show options for")
			}

			return cmd.Run(context.Background(), args[0])
		},
	}

	optionsCmd.Flags().BoolVar(&cmd.Hidden, "hidden", false, "If true, will also show hidden options.")
	optionsCmd.Flags().StringVar(&cmd.Output, "output", "plain", "The output format to use. Can be json or plain")
	return optionsCmd
}

type optionWithValue struct {
	provider2.ProviderOption `json:",inline"`

	Value string `json:"value,omitempty"`
}

// Run runs the command logic
func (cmd *OptionsCmd) Run(ctx context.Context, providerName string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context)
	if err != nil {
		return err
	}

	provider, err := workspace.FindProvider(devPodConfig, providerName, log.Default)
	if err != nil {
		return err
	}

	entryOptions := provider.Options
	if entryOptions == nil {
		entryOptions = map[string]config.OptionValue{}
	}

	if cmd.Output == "plain" {
		tableEntries := [][]string{}
		for optionName, entry := range provider.Config.Options {
			if !cmd.Hidden && entry.Hidden {
				continue
			}

			tableEntries = append(tableEntries, []string{
				optionName,
				strconv.FormatBool(entry.Required),
				entry.Description,
				entry.Default,
				entryOptions[optionName].Value,
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

		out, err := json.Marshal(options)
		if err != nil {
			return err
		}
		fmt.Print(string(out))
	} else {
		return fmt.Errorf("unexpected output format, choose either json or plain. Got %s", cmd.Output)
	}

	return nil
}
