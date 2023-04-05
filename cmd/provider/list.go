package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/log/table"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/spf13/cobra"
	"sort"
	"strconv"
)

// ListCmd holds the list cmd flags
type ListCmd struct {
	flags.GlobalFlags

	Output string
	Used   bool
}

// NewListCmd creates a new command
func NewListCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &ListCmd{
		GlobalFlags: *flags,
	}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available providers",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background())
		},
	}

	listCmd.Flags().StringVar(&cmd.Output, "output", "plain", "The output format to use. Can be json or plain")
	listCmd.Flags().BoolVar(&cmd.Used, "used", false, "If enabled, will only show used providers")
	return listCmd
}

// Run runs the command logic
func (cmd *ListCmd) Run(ctx context.Context) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	providers, err := workspace.LoadAllProviders(devPodConfig, log.Default.ErrorStreamOnly())
	if err != nil {
		return err
	}

	configuredProviders := devPodConfig.Current().Providers
	if configuredProviders == nil {
		configuredProviders = map[string]*config.ProviderConfig{}
	}

	if cmd.Output == "plain" {
		tableEntries := [][]string{}
		for _, entry := range providers {
			if cmd.Used && configuredProviders[entry.Config.Name] == nil {
				continue
			}

			tableEntries = append(tableEntries, []string{
				entry.Config.Name,
				entry.Config.Version,
				strconv.FormatBool(devPodConfig.Current().DefaultProvider == entry.Config.Name),
				strconv.FormatBool(entry.State != nil),
				entry.Config.Description,
			})
		}
		sort.SliceStable(tableEntries, func(i, j int) bool {
			return tableEntries[i][0] < tableEntries[j][0]
		})

		table.PrintTable(log.Default, []string{
			"Name",
			"Version",
			"Default",
			"Configured",
			"Description",
		}, tableEntries)
	} else if cmd.Output == "json" {
		out, err := json.Marshal(providers)
		if err != nil {
			return err
		}
		fmt.Print(string(out))
	} else {
		return fmt.Errorf("unexpected output format, choose either json or plain. Got %s", cmd.Output)
	}

	return nil
}
