// Package ts provides the tailscale commands within the DevPod CLI useful for debugging the network.
// These file were copied from the tailscale project https://github.com/tailscale/tailscale/tree/v1.78.3/cmd/tailscale/cli
// and modified to work with our `pkg/tailscale` package that connects tsnet to loft's control plane & custom DERP.
package ts

import (
	"strings"

	"github.com/loft-sh/devpod/cmd/agent"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/spf13/cobra"
)

func NewTailscaleCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	tsCmd := &cobra.Command{
		Use:   "ts",
		Short: "DevPod tailscale commands",
		Long: strings.TrimSpace(`
			Tailscale client commands that are useful for debugging have been ported into DevPod CLI.
			For more information about usage, refer to https://tailscale.com/kb/1080/cli
		`),
		PersistentPreRunE: func(cobraCmd *cobra.Command, args []string) error {
			return agent.AgentPersistentPreRunE(cobraCmd, args, globalFlags)
		},
		Hidden: true,
	}

	tsCmd.AddCommand(NewNetcheckCmd())
	tsCmd.AddCommand(NewStatusCmd())
	tsCmd.AddCommand(NewPingCmd())
	tsCmd.AddCommand(NewMetricsCmd())

	return tsCmd
}
