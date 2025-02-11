//go:build windows

package container

import (
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/spf13/cobra"
)

func NewSetupContainerCmd(flags *flags.GlobalFlags) *cobra.Command {
	panic("Not implemented for windows, this should never be called!")
	return nil
}
