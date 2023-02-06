package agent

import (
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/spf13/cobra"
)

// NewAgentCmd returns a new root command
func NewAgentCmd(flags *flags.GlobalFlags) *cobra.Command {
	agentCmd := &cobra.Command{
		Use:    "agent",
		Short:  "DevPod Agent",
		Hidden: true,
	}

	agentCmd.AddCommand(NewSSHServerCmd())
	agentCmd.AddCommand(NewUpCmd(flags))
	agentCmd.AddCommand(NewContainerTunnelCmd())
	return agentCmd
}
