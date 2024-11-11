package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// ExportCmd holds the export cmd flags
type ExportCmd struct {
	*flags.GlobalFlags
}

// NewExportCmd creates a new command
func NewExportCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &ExportCmd{
		GlobalFlags: flags,
	}
	exportCmd := &cobra.Command{
		Use:   "export [flags] [workspace-path|workspace-name]",
		Short: "Exports a workspace configuration",
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()
			devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, devPodConfig, args)
		},
	}

	return exportCmd
}

// Run runs the command logic
func (cmd *ExportCmd) Run(ctx context.Context, devPodConfig *config.Config, args []string) error {
	// try to load workspace
	logger := log.Default.ErrorStreamOnly()
	client, err := workspace2.Get(ctx, devPodConfig, args, false, logger)
	if err != nil {
		return err
	}

	// export workspace
	exportConfig, err := exportWorkspace(devPodConfig, client.WorkspaceConfig())
	if err != nil {
		return err
	}

	// marshal config
	out, err := json.Marshal(exportConfig)
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}

func exportWorkspace(devPodConfig *config.Config, workspaceConfig *provider.Workspace) (*provider.ExportConfig, error) {
	var err error

	// create return config
	retConfig := &provider.ExportConfig{}

	// export workspace
	retConfig.Workspace, err = provider.ExportWorkspace(workspaceConfig.Context, workspaceConfig.ID)
	if err != nil {
		return nil, fmt.Errorf("export workspace config: %w", err)
	}

	// has machine?
	if workspaceConfig.Machine.ID != "" {
		retConfig.Machine, err = provider.ExportMachine(workspaceConfig.Context, workspaceConfig.Machine.ID)
		if err != nil {
			return nil, fmt.Errorf("export machine config: %w", err)
		}
	}

	// export provider
	retConfig.Provider, err = provider.ExportProvider(devPodConfig, workspaceConfig.Context, workspaceConfig.Provider.Name)
	if err != nil {
		return nil, fmt.Errorf("export provider config: %w", err)
	}

	return retConfig, nil
}
