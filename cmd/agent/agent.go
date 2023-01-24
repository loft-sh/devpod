package agent

import (
	"github.com/spf13/cobra"
)

// NewAgentCmd returns a new root command
func NewAgentCmd() *cobra.Command {
	agentCmd := &cobra.Command{
		Use:    "agent",
		Short:  "DevPod Agent",
		Hidden: true,
	}

	agentCmd.AddCommand(NewSSHServerCmd())
	agentCmd.AddCommand(NewUpCmd())
	agentCmd.AddCommand(NewContainerTunnelCmd())
	return agentCmd
}
