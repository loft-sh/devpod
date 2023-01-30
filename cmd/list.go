package cmd

import (
	"context"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/log/table"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"time"
)

// ListCmd holds the configuration
type ListCmd struct {
}

// NewListCmd creates a new destroy command
func NewListCmd() *cobra.Command {
	cmd := &ListCmd{}
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
	workspaceDir, err := config.GetWorkspacesDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(workspaceDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	tableEntries := [][]string{}
	for _, entry := range entries {
		workspaceConfig, err := config.LoadWorkspaceConfig(entry.Name())
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

	table.PrintTable(log.Default, []string{
		"Name",
		"Source",
		"Provider",
		"Age",
	}, tableEntries)
	return nil
}
