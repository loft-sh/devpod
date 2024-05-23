package provider

import (
	"os"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/cmd/pro/provider/list"
	"github.com/loft-sh/devpod/pkg/loft"
	"github.com/loft-sh/devpod/pkg/loft/client"

	"github.com/spf13/cobra"
)

// NewProProviderCmd creates a new cobra command
func NewProProviderCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	c := &cobra.Command{
		Use:    "provider",
		Short:  "DevPod Pro provider commands",
		Args:   cobra.NoArgs,
		Hidden: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if (globalFlags.Config == "" || globalFlags.Config == client.DefaultCacheConfig) && os.Getenv("LOFT_CONFIG") != "" {
				globalFlags.Config = os.Getenv(loft.ConfigEnv)
			}
		},
	}

	c.AddCommand(list.NewListCmd(globalFlags))
	c.AddCommand(NewUpCmd(globalFlags))
	c.AddCommand(NewStopCmd(globalFlags))
	c.AddCommand(NewSshCmd(globalFlags))
	c.AddCommand(NewStatusCmd(globalFlags))
	c.AddCommand(NewDeleteCmd(globalFlags))
	c.AddCommand(NewRebuildCmd(globalFlags))
	return c
}
