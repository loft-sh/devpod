package cmd

import (
	"github.com/spf13/cobra"
)

// NewDockerCmd returns a new root command
func NewDockerCmd() *cobra.Command {
	dockerCmd := &cobra.Command{
		Use:   "docker",
		Short: "docker Provider commands",
	}

	dockerCmd.AddCommand(NewInitCmd())
	dockerCmd.AddCommand(NewCreateCmd())
	dockerCmd.AddCommand(NewDeleteCmd())
	dockerCmd.AddCommand(NewStatusCmd())
	dockerCmd.AddCommand(NewTunnelCmd())
	dockerCmd.AddCommand(NewStartCmd())
	dockerCmd.AddCommand(NewStopCmd())
	return dockerCmd
}
