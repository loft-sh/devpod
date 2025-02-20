//go:build windows

package container

import (
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/spf13/cobra"
)

func NewSetupContainerCmd(flags *flags.GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Sets up a container",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			panic("Windows Containers are not supported")
		},
	}
}
