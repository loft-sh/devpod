package ide

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/ide/ideparse"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/log/table"
	"github.com/spf13/cobra"
)

// OptionsCmd holds the options cmd flags
type OptionsCmd struct {
	flags.GlobalFlags

	Output string
}

// NewOptionsCmd creates a new command
func NewOptionsCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &OptionsCmd{
		GlobalFlags: *flags,
	}
	optionsCmd := &cobra.Command{
		Use:   "options",
		Short: "List ide options",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("please specify the ide")
			}

			return cmd.Run(context.Background(), args[0])
		},
	}

	optionsCmd.Flags().StringVar(&cmd.Output, "output", "plain", "The output format to use. Can be json or plain")
	return optionsCmd
}

type optionWithValue struct {
	ide.Option `json:",inline"`

	Value string `json:"value,omitempty"`
}

// Run runs the command logic
func (cmd *OptionsCmd) Run(ctx context.Context, ide string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	values := devPodConfig.IDEOptions(ide)
	ideOptions, err := ideparse.GetIDEOptions(ide)
	if err != nil {
		return err
	}

	if cmd.Output == "plain" {
		tableEntries := [][]string{}
		for optionName, entry := range ideOptions {
			value := values[optionName].Value
			tableEntries = append(tableEntries, []string{
				optionName,
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
		for optionName, entry := range ideOptions {
			options[optionName] = optionWithValue{
				Option: entry,
				Value:  values[optionName].Value,
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
