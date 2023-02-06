package provider

import (
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/cmd/provider/call"
	"github.com/spf13/cobra"
)

// NewProviderCmd returns a new root command
func NewProviderCmd(flags *flags.GlobalFlags) *cobra.Command {
	providerCmd := &cobra.Command{
		Use:   "provider",
		Short: "DevPod Provider commands",
	}

	providerCmd.AddCommand(call.NewCallCmd())
	providerCmd.AddCommand(NewListCmd(flags))
	providerCmd.AddCommand(NewUseCmd(flags))
	providerCmd.AddCommand(NewOptionsCmd(flags))
	return providerCmd
}
