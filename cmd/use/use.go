package use

import (
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/cmd/provider"
	"github.com/spf13/cobra"
)

// NewUseCmd returns a new root command
func NewUseCmd(flags *flags.GlobalFlags) *cobra.Command {
	useCmd := &cobra.Command{
		Use:   "use",
		Short: "Use DevPod resources",
	}

	useProviderCmd := provider.NewUseCmd(flags)
	useProviderCmd.Use = "provider"
	useCmd.AddCommand(useProviderCmd)
	return useCmd
}
