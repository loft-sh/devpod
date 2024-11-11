package watch

import (
	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/spf13/cobra"
)

// NewCmd creates a new cobra command
func NewCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	c := &cobra.Command{
		Use:    "watch",
		Short:  "DevPod Pro Provider watch commands",
		Args:   cobra.NoArgs,
		Hidden: true,
	}

	c.AddCommand(NewWorkspacesCmd(globalFlags))

	return c
}
