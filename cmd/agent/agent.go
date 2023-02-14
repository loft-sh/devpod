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

	agentCmd.AddCommand(NewUpCmd(flags))
	agentCmd.AddCommand(NewContainerTunnelCmd())
	agentCmd.AddCommand(NewWatchCmd())
	agentCmd.AddCommand(NewDeleteCmd(flags))
	agentCmd.AddCommand(NewStopCmd(flags))
	agentCmd.AddCommand(NewStartCmd(flags))
	agentCmd.AddCommand(NewStatusCmd(flags))
	agentCmd.AddCommand(NewOpenVSCodeCmd())
	return agentCmd
}
