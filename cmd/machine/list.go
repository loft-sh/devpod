package machine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Lists existing machines",
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

	machineDir, err := provider.GetMachinesDir(devPodConfig.DefaultContext)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(machineDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if cmd.Output == "plain" {
		tableEntries := [][]string{}
		for _, entry := range entries {
			machineConfig, err := provider.LoadMachineConfig(devPodConfig.DefaultContext, entry.Name())
			if err != nil {
				return errors.Wrap(err, "load machine config")
			}

			tableEntries = append(tableEntries, []string{
				machineConfig.ID,
				machineConfig.Provider.Name,
				time.Since(machineConfig.CreationTimestamp.Time).Round(1 * time.Second).String(),
			})
		}
		sort.SliceStable(tableEntries, func(i, j int) bool {
			return tableEntries[i][0] < tableEntries[j][0]
		})

		table.PrintTable(log.Default, []string{
			"Name",
			"Provider",
			"Age",
		}, tableEntries)
	} else if cmd.Output == "json" {
		tableEntries := []*provider.Machine{}
		for _, entry := range entries {
			machineConfig, err := provider.LoadMachineConfig(devPodConfig.DefaultContext, entry.Name())
			if err != nil {
				return errors.Wrap(err, "load machine config")
			}

			tableEntries = append(tableEntries, machineConfig)
		}
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
