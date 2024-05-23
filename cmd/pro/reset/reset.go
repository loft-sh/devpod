package reset

import (
	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/spf13/cobra"
)

// NewResetCmd creates a new cobra command
func NewResetCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	c := &cobra.Command{
		Use:   "reset",
		Short: "Reset configuration",
		Args:  cobra.NoArgs,
	}

	c.AddCommand(NewPasswordCmd(globalFlags))
	return c
}
