package provider

import (
	"os"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/cmd/pro/provider/create"
	"github.com/loft-sh/devpod/cmd/pro/provider/get"
	"github.com/loft-sh/devpod/cmd/pro/provider/list"
	"github.com/loft-sh/devpod/cmd/pro/provider/update"
	"github.com/loft-sh/devpod/cmd/pro/provider/watch"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/log"

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
				globalFlags.Config = os.Getenv(platform.ConfigEnv)
			}

			log.Default.SetFormat(log.JSONFormat)
		},
	}

	c.AddCommand(list.NewCmd(globalFlags))
	c.AddCommand(watch.NewCmd(globalFlags))
	c.AddCommand(create.NewCmd(globalFlags))
	c.AddCommand(get.NewCmd(globalFlags))
	c.AddCommand(update.NewCmd(globalFlags))
	c.AddCommand(NewHealthCmd(globalFlags))

	c.AddCommand(NewUpCmd(globalFlags))
	c.AddCommand(NewStopCmd(globalFlags))
	c.AddCommand(NewSshCmd(globalFlags))
	c.AddCommand(NewStatusCmd(globalFlags))
	c.AddCommand(NewDeleteCmd(globalFlags))
	c.AddCommand(NewRebuildCmd(globalFlags))
	return c
}
