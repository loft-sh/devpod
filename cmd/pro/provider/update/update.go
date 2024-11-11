package update

import (
	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/spf13/cobra"
)

// NewCmd creates a new cobra command
func NewCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	c := &cobra.Command{
		Use:    "update",
		Short:  "DevPod Pro Provider update commands",
		Args:   cobra.NoArgs,
		Hidden: true,
	}

	c.AddCommand(NewWorkspaceCmd(globalFlags))

	return c
}
