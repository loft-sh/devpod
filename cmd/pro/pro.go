package pro

import (
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/spf13/cobra"
)

// NewProCmd returns a new command
func NewProCmd(flags *flags.GlobalFlags) *cobra.Command {
	proCmd := &cobra.Command{
		Use:   "pro",
		Short: "DevPod Pro commands",
	}

	proCmd.AddCommand(NewLoginCmd(flags))
	proCmd.AddCommand(NewListCmd(flags))
	proCmd.AddCommand(NewDeleteCmd(flags))
	return proCmd
}
