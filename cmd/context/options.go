package context

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/spf13/cobra"
)

// OptionsCmd holds the options cmd flags
type OptionsCmd struct {
	*flags.GlobalFlags

	Output string
}

// NewOptionsCmd creates a new command
func NewOptionsCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &OptionsCmd{
		GlobalFlags: flags,
	}
	optionsCmd := &cobra.Command{
		Use:   "options",
		Short: "Show options of a context",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	optionsCmd.Flags().StringVar(&cmd.Output, "output", "plain", "The output format to use. Can be json or plain")
	return optionsCmd
}

type optionWithValue struct {
	config.ContextOption `json:",inline"`

	Value string `json:"value,omitempty"`
}

// Run runs the command logic
func (cmd *OptionsCmd) Run(ctx context.Context, args []string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, "")
	if err != nil {
		return err
	}

	entryOptions := devPodConfig.Current().Options
	if entryOptions == nil {
		entryOptions = map[string]config.OptionValue{}
	}

	if cmd.Output == "plain" {
		tableEntries := [][]string{}
		for _, entry := range config.ContextOptions {
			value := entryOptions[entry.Name].Value

			tableEntries = append(tableEntries, []string{
				entry.Name,
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
			"Description",
			"Default",
			"Value",
		}, tableEntries)
	} else if cmd.Output == "json" {
		options := map[string]optionWithValue{}
		for _, entry := range config.ContextOptions {
			options[entry.Name] = optionWithValue{
				ContextOption: entry,
				Value:         entryOptions[entry.Name].Value,
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
