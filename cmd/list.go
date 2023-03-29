package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/log/table"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/spf13/cobra"
	"os"
	"sort"
	"time"
)

// ListCmd holds the configuration
type ListCmd struct {
	*flags.GlobalFlags

	Output string
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

	listCmd.Flags().StringVar(&cmd.Output, "output", "plain", "The output format to use. Can be json or plain")
	return listCmd
}

// Run runs the command logic
func (cmd *ListCmd) Run(ctx context.Context) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	workspaces, err := listWorkspaces(devPodConfig, log.Default)
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
			workspaceConfig, err := provider2.LoadWorkspaceConfig(devPodConfig.DefaultContext, entry.ID)
			if err != nil {
				log.Default.ErrorStreamOnly().Warnf("Couldn't load workspace %s: %v", entry.ID, err)
				continue
			}

			tableEntries = append(tableEntries, []string{
				workspaceConfig.ID,
				workspaceConfig.Source.String(),
				workspaceConfig.Machine.ID,
				workspaceConfig.Provider.Name,
				time.Since(workspaceConfig.LastUsedTimestamp.Time).Round(1 * time.Second).String(),
				time.Since(workspaceConfig.CreationTimestamp.Time).Round(1 * time.Second).String(),
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
			"Last Used",
			"Age",
		}, tableEntries)
	} else {
		return fmt.Errorf("unexpected output format, choose either json or plain. Got %s", cmd.Output)
	}

	return nil
}

func listWorkspaces(devPodConfig *config.Config, log log.Logger) ([]*provider2.Workspace, error) {
	workspaceDir, err := provider2.GetWorkspacesDir(devPodConfig.DefaultContext)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(workspaceDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	retWorkspaces := []*provider2.Workspace{}
	for _, entry := range entries {
		workspaceConfig, err := provider2.LoadWorkspaceConfig(devPodConfig.DefaultContext, entry.Name())
		if err != nil {
			log.ErrorStreamOnly().Warnf("Couldn't load workspace %s: %v", entry.Name(), err)
			continue
		}

		retWorkspaces = append(retWorkspaces, workspaceConfig)
	}

	return retWorkspaces, nil
}
