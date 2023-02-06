package provider

import (
	"context"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/log/table"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/spf13/cobra"
	"strconv"
)

// ListCmd holds the list cmd flags
type ListCmd struct {
	flags.GlobalFlags
}

// NewListCmd creates a new destroy command
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

	return listCmd
}

// Run runs the command logic
func (cmd *ListCmd) Run(ctx context.Context) error {
	devPodConfig, err := config.LoadConfig(cmd.Context)
	if err != nil {
		return err
	}

	providers, err := workspace.LoadAllProviders(devPodConfig, log.Default)
	if err != nil {
		return err
	}

	tableEntries := [][]string{}
	for _, entry := range providers {
		tableEntries = append(tableEntries, []string{
			entry.Provider.Name(),
			strconv.FormatBool(devPodConfig.Contexts[devPodConfig.DefaultContext].DefaultProvider == entry.Provider.Name()),
			entry.Provider.Description(),
		})
	}

	table.PrintTable(log.Default, []string{
		"Name",
		"Default",
		"Description",
	}, tableEntries)
	return nil
}
