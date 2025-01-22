package ts

import (
	"github.com/loft-sh/devpod/cmd/agent"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/spf13/cobra"
)

// NewHelperCmd returns a new command
func NewTailscaleCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	helperCmd := &cobra.Command{
		Use:   "ts",
		Short: "DevPod tailscale Commands",
		PersistentPreRunE: func(cobraCmd *cobra.Command, args []string) error {
			return agent.AgentPersistentPreRunE(cobraCmd, args, globalFlags)
		},
		Hidden: true,
	}

	helperCmd.AddCommand(NewNetcheckCmd(globalFlags))
	helperCmd.AddCommand(NewStatusCmd(globalFlags))
	helperCmd.AddCommand(NewPingCmd(globalFlags))

	return helperCmd
}
