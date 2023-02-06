package call

import (
	dockercmd "github.com/loft-sh/devpod/providers/docker/cmd"
	gcloudcmd "github.com/loft-sh/devpod/providers/gcloud/cmd"
	"github.com/spf13/cobra"
)

// NewCallCmd returns a new root command
func NewCallCmd() *cobra.Command {
	providerCmd := &cobra.Command{
		Use:    "call",
		Short:  "Call in-built provider commands",
		Hidden: true,
	}

	providerCmd.AddCommand(dockercmd.NewDockerCmd())
	providerCmd.AddCommand(gcloudcmd.NewGCloudCmd())
	return providerCmd
}
