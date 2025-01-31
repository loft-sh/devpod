// Package tailscale embeds the tailscale CLI into the DevPod CLI. This allows users to run tailscale commands by prefixing them
// with 'devpod tailscale'.
package ts

import (
	"os"

	"github.com/spf13/cobra"
	"tailscale.com/cmd/tailscaled/cli"
)

func NewTailscaledCmd() *cobra.Command {
	tsCmd := &cobra.Command{
		Use:                "tailscaled",
		Short:              "DevPod tailscaled wrapper, supporting all the usual flags 'devpod tailscaled'",
		Hidden:             true,
		DisableFlagParsing: true,
		Run: func(_ *cobra.Command, _ []string) {
			// Defer to the tailscale CLI after stripping the devpod tailscale prefix
			cli.Run(os.Args[2:])
		},
	}
	return tsCmd
}
