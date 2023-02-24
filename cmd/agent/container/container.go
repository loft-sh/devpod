package container

import (
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/spf13/cobra"
)

// NewContainerCmd returns a new command
func NewContainerCmd(flags *flags.GlobalFlags) *cobra.Command {
	containerCmd := &cobra.Command{
		Use:   "container",
		Short: "Container commands",
	}

	containerCmd.AddCommand(NewSetupContainerCmd())
	containerCmd.AddCommand(NewDaemonCmd())
	containerCmd.AddCommand(NewVSCodeAsyncCmd())
	return containerCmd
}
