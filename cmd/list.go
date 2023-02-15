package cmd

import (
	"context"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/log/table"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"sort"
	"time"
)

// ListCmd holds the configuration
type ListCmd struct {
	*flags.GlobalFlags
}

// NewListCmd creates a new destroy command
func NewListCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &ListCmd{
		GlobalFlags: flags,
	}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists existing workspaces",
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

	workspaceDir, err := config.GetWorkspacesDir(devPodConfig.DefaultContext)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(workspaceDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	tableEntries := [][]string{}
	for _, entry := range entries {
		workspaceConfig, err := config.LoadWorkspaceConfig(devPodConfig.DefaultContext, entry.Name())
		if err != nil {
			return errors.Wrap(err, "load workspace config")
		}

		tableEntries = append(tableEntries, []string{
			workspaceConfig.ID,
			workspaceConfig.Source.String(),
			workspaceConfig.Provider.Name,
			time.Since(workspaceConfig.CreationTimestamp.Time).Round(1 * time.Second).String(),
		})
	}
	sort.SliceStable(tableEntries, func(i, j int) bool {
		return tableEntries[i][0] < tableEntries[j][0]
	})

	table.PrintTable(log.Default, []string{
		"Name",
		"Source",
		"Provider",
		"Age",
	}, tableEntries)
	return nil
}
