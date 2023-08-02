package engine

import (
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/spf13/cobra"
)

// NewEngineCmd returns a new command
func NewEngineCmd(flags *flags.GlobalFlags) *cobra.Command {
	engineCmd := &cobra.Command{
		Use:   "engine",
		Short: "DevPod Engine commands",
	}

	engineCmd.AddCommand(NewLoginCmd(flags))
	return engineCmd
}
