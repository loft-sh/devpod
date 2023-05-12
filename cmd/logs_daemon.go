package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/spf13/cobra"
)

// LogsDaemonCmd holds the configuration
type LogsDaemonCmd struct {
	*flags.GlobalFlags
}

// NewLogsDaemonCmd creates a new destroy command
func NewLogsDaemonCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &LogsDaemonCmd{
		GlobalFlags: flags,
	}
	startCmd := &cobra.Command{
		Use:   "logs-daemon",
		Short: "Prints the daemon logs on the machine",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	return startCmd
}

// Run runs the command logic
func (cmd *LogsDaemonCmd) Run(ctx context.Context, args []string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	workspaceClient, err := workspace.GetWorkspace(devPodConfig, args, false, log.Default)
	if err != nil {
		return err
	} else if workspaceClient.WorkspaceConfig().Machine.ID == "" {
		return fmt.Errorf("selected workspace is not a machine provider, there is not daemon running")
	}

	_, agentInfo, err := workspaceClient.AgentInfo()
	if err != nil {
		return err
	}

	command := fmt.Sprintf("%s agent workspace logs-daemon --context '%s' --id '%s'", workspaceClient.AgentPath(), workspaceClient.Context(), workspaceClient.Workspace())
	if agentInfo.Agent.DataPath != "" {
		command += fmt.Sprintf(" --agent-dir '%s'", agentInfo.Agent.DataPath)
	}

	// read daemon logs
	return workspaceClient.Command(ctx, client.CommandOptions{
		Command: command,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	})
}
