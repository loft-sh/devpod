package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/spf13/cobra"
	"os"
)

// CommandCmd holds the cmd flags
type CommandCmd struct{}

// NewCommandCmd defines a command
func NewCommandCmd() *cobra.Command {
	cmd := &CommandCmd{}
	commandCmd := &cobra.Command{
		Use:   "command",
		Short: "Command an instance",
		RunE: func(_ *cobra.Command, args []string) error {
			gcloudProvider, err := newProvider(log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), gcloudProvider, provider2.FromEnvironment(), log.Default)
		},
	}

	return commandCmd
}

// Run runs the command logic
func (cmd *CommandCmd) Run(ctx context.Context, provider *gcloudProvider, workspace *provider2.Workspace, log log.Logger) error {
	command := os.Getenv(provider2.CommandEnv)
	if command == "" {
		return fmt.Errorf("command is empty")
	}

	// via cmd
	name := getName(workspace)
	args := []string{
		"compute",
		"ssh",
		name,
		"--project=" + provider.Config.Project,
		"--zone=" + provider.Config.Zone,
		"--command",
		command,
	}

	return provider.runCommand(ctx, args, os.Stdout, os.Stderr, os.Stdin)
}
