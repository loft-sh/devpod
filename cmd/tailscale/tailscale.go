// Package tailscale embeds the tailscale CLI into the DevPod CLI. This allows users to run tailscale commands by prefixing them
// with 'devpod tailscale'.
package ts

import (
	"os"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/spf13/cobra"
	"tailscale.com/cmd/tailscale/cli"
)

func NewTailscaleCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	tsCmd := &cobra.Command{
		Use:    "tailscale",
		Short:  "DevPod tailscale commands, simply prefix a tailscale CLI command with 'devpod tailscale'",
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			// Defer to the tailscale CLI after stripping the devpod tailscale prefix
			return cli.Run(os.Args[2:])
		},
	}
	return tsCmd
}
