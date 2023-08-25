package provider

import (
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/spf13/cobra"
)

// NewProviderCmd returns a new root command
func NewProviderCmd(flags *flags.GlobalFlags) *cobra.Command {
	providerCmd := &cobra.Command{
		Use:   "provider",
		Short: "DevPod Provider commands",
	}

	providerCmd.AddCommand(NewListCmd(flags))
	providerCmd.AddCommand(NewListAvailableCmd(flags))
	providerCmd.AddCommand(NewUseCmd(flags))
	providerCmd.AddCommand(NewOptionsCmd(flags))
	providerCmd.AddCommand(NewDeleteCmd(flags))
	providerCmd.AddCommand(NewAddCmd(flags))
	providerCmd.AddCommand(NewUpdateCmd(flags))
	providerCmd.AddCommand(NewSetOptionsCmd(flags))
	return providerCmd
}
