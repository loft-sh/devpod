package pro

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/spf13/cobra"
)

// ListCmd holds the list cmd flags
type ListCmd struct {
	flags.GlobalFlags

	Output string
}

// NewListCmd creates a new command
func NewListCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &ListCmd{
		GlobalFlags: *flags,
	}
	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List available pro instances",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background())
		},
	}

	listCmd.Flags().StringVar(&cmd.Output, "output", "plain", "The output format to use. Can be json or plain")
	return listCmd
}

// Run runs the command logic
func (cmd *ListCmd) Run(ctx context.Context) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	proInstances, err := workspace.ListProInstances(devPodConfig, log.Default)
	if err != nil {
		return err
	}

	if cmd.Output == "plain" {
		tableEntries := [][]string{}
		for _, proInstance := range proInstances {
			tableEntries = append(tableEntries, []string{
				proInstance.ID,
				proInstance.URL,
				time.Since(proInstance.CreationTimestamp.Time).Round(1 * time.Second).String(),
			})
		}
		sort.SliceStable(tableEntries, func(i, j int) bool {
			return tableEntries[i][0] < tableEntries[j][0]
		})

		table.PrintTable(log.Default, []string{
			"Name",
			"Url",
			"Age",
		}, tableEntries)
	} else if cmd.Output == "json" {
		tableEntries := []*provider.ProInstance{}
		tableEntries = append(tableEntries, proInstances...)
		sort.SliceStable(tableEntries, func(i, j int) bool {
			return tableEntries[i].ID < tableEntries[j].ID
		})
		out, err := json.Marshal(tableEntries)
		if err != nil {
			return err
		}
		fmt.Print(string(out))
	} else {
		return fmt.Errorf("unexpected output format, choose either json or plain. Got %s", cmd.Output)
	}

	return nil
}
