package provider

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/log/table"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/spf13/cobra"
)

// OptionsCmd holds the options cmd flags
type OptionsCmd struct {
	flags.GlobalFlags
}

// NewOptionsCmd creates a new destroy command
func NewOptionsCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &OptionsCmd{
		GlobalFlags: *flags,
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

	return optionsCmd
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

	tableEntries := [][]string{}
	for optionName, entry := range provider.Provider.Options() {
		if entry.Hidden {
			continue
		}

		entryOptions := provider.Options
		if entryOptions == nil {
			entryOptions = map[string]provider2.OptionValue{}
		}

		tableEntries = append(tableEntries, []string{
			optionName,
			entry.Description,
			entry.Default,
			entryOptions[optionName].Value,
		})
	}

	table.PrintTable(log.Default, []string{
		"Name",
		"Description",
		"Default",
		"Value",
	}, tableEntries)
	return nil
}
