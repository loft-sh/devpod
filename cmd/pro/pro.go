package pro

import (
	"os"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/cmd/pro/add"
	proflags "github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/cmd/pro/provider"
	"github.com/loft-sh/devpod/cmd/pro/reset"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewProCmd returns a new command
func NewProCmd(flags *flags.GlobalFlags, streamLogger *log.StreamLogger) *cobra.Command {
	globalFlags := &proflags.GlobalFlags{GlobalFlags: flags}
	proCmd := &cobra.Command{
		Use:           "pro",
		Short:         "DevPod Pro commands",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		PersistentPreRunE: func(c *cobra.Command, _ []string) error {
			globalFlags = proflags.SetGlobalFlags(c.PersistentFlags())
			if flags.Silent {
				streamLogger.SetLevel(logrus.FatalLevel)
			}
			if flags.Debug {
				streamLogger.SetLevel(logrus.DebugLevel)
			}

			if os.Getenv("DEVPOD_DEBUG") == "true" {
				log.Default.SetLevel(logrus.DebugLevel)
			}
			if flags.LogOutput == "json" {
				log.Default.SetFormat(log.JSONFormat)
			}

			return nil
		},
	}

	proCmd.AddCommand(NewLoginCmd(globalFlags))
	proCmd.AddCommand(NewListCmd(globalFlags))
	proCmd.AddCommand(NewDeleteCmd(globalFlags))
	proCmd.AddCommand(NewImportCmd(globalFlags))
	proCmd.AddCommand(NewStartCmd(globalFlags))
	proCmd.AddCommand(NewRebuildCmd(globalFlags))
	proCmd.AddCommand(NewSleepCmd(globalFlags))
	proCmd.AddCommand(NewWakeupCmd(globalFlags))
	proCmd.AddCommand(reset.NewResetCmd(globalFlags))
	proCmd.AddCommand(provider.NewProProviderCmd(globalFlags))
	proCmd.AddCommand(add.NewAddCmd(globalFlags))
	proCmd.AddCommand(NewWatchWorkspacesCmd(globalFlags))
	proCmd.AddCommand(NewSelfCmd(globalFlags))
	proCmd.AddCommand(NewVersionCmd(globalFlags))
	proCmd.AddCommand(NewListProjectsCmd(globalFlags))
	proCmd.AddCommand(NewListWorkspacesCmd(globalFlags))
	proCmd.AddCommand(NewListTemplatesCmd(globalFlags))
	proCmd.AddCommand(NewListClustersCmd(globalFlags))
	proCmd.AddCommand(NewCreateWorkspaceCmd(globalFlags))
	proCmd.AddCommand(NewUpdateWorkspaceCmd(globalFlags))
	proCmd.AddCommand(NewCheckHealthCmd(globalFlags))
	proCmd.AddCommand(NewCheckUpdateCmd(globalFlags))
	proCmd.AddCommand(NewUpdateProviderCmd(globalFlags))
	return proCmd
}
