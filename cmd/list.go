package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/spf13/cobra"
)

// ListCmd holds the configuration
type ListCmd struct {
	*flags.GlobalFlags

	Output  string
	SkipPro bool
}

// NewListCmd creates a new destroy command
func NewListCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &ListCmd{
		GlobalFlags: flags,
	}
	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lists existing workspaces",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("no arguments are allowed for this command")
			}

			return cmd.Run(context.Background())
		},
	}

	listCmd.Flags().StringVar(&cmd.Output, "output", "plain", "The output format to use. Can be json or plain")
	listCmd.Flags().BoolVar(&cmd.SkipPro, "skip-pro", false, "Don't list pro workspaces")
	return listCmd
}

// Run runs the command logic
func (cmd *ListCmd) Run(ctx context.Context) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	workspaces, err := workspace.List(ctx, devPodConfig, cmd.SkipPro, log.Default)
	if err != nil {
		return err
	}

	if cmd.Output == "json" {
		sort.SliceStable(workspaces, func(i, j int) bool {
			return workspaces[i].ID < workspaces[j].ID
		})
		out, err := json.Marshal(workspaces)
		if err != nil {
			return err
		}
		fmt.Print(string(out))
	} else if cmd.Output == "plain" {
		tableEntries := [][]string{}
		for _, entry := range workspaces {
			tableEntries = append(tableEntries, []string{
				entry.ID,
				entry.Source.String(),
				entry.Machine.ID,
				entry.Provider.Name,
				entry.IDE.Name,
				time.Since(entry.LastUsedTimestamp.Time).Round(1 * time.Second).String(),
				time.Since(entry.CreationTimestamp.Time).Round(1 * time.Second).String(),
				fmt.Sprintf("%t", entry.IsPro()),
			})
		}

		sort.SliceStable(tableEntries, func(i, j int) bool {
			return tableEntries[i][0] < tableEntries[j][0]
		})
		table.PrintTable(log.Default, []string{
			"Name",
			"Source",
			"Machine",
			"Provider",
			"IDE",
			"Last Used",
			"Age",
			"Pro",
		}, tableEntries)
	} else {
		return fmt.Errorf("unexpected output format, choose either json or plain. Got %s", cmd.Output)
	}

	return nil
}
