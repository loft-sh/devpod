// Package ts provides the tailscale commands within the DevPod CLI useful for debugging the network.
// These file were copied from the tailscale project https://github.com/tailscale/tailscale/tree/v1.78.3/cmd/tailscale/cli
// and modified to work with our `pkg/tailscale` package that connects tsnet to loft's control plane & custom DERP.
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
