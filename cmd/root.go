package cmd

import (
	"github.com/loft-sh/devpod/cmd/agent"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/cmd/provider"
	log2 "github.com/loft-sh/devpod/pkg/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var globalFlags *flags.GlobalFlags

// NewRootCmd returns a new root command
func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "devpod",
		Short:         "DevPod",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cobraCmd *cobra.Command, args []string) error {
			if globalFlags.Silent {
				log2.Default.SetLevel(logrus.FatalLevel)
			} else if globalFlags.Debug {
				log2.Default.SetLevel(logrus.DebugLevel)
			}

			return nil
		},
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// build the root command
	rootCmd := BuildRoot()

	// execute command
	err := rootCmd.Execute()
	if err != nil {
		if globalFlags.Debug {
			log2.Default.Fatalf("%+v", err)
		} else {
			log2.Default.Fatal(err)
		}
	}
}

// BuildRoot creates a new root command from the
func BuildRoot() *cobra.Command {
	rootCmd := NewRootCmd()
	persistentFlags := rootCmd.PersistentFlags()
	globalFlags = flags.SetGlobalFlags(persistentFlags)

	rootCmd.AddCommand(agent.NewAgentCmd(globalFlags))
	rootCmd.AddCommand(provider.NewProviderCmd(globalFlags))
	rootCmd.AddCommand(NewUpCmd(globalFlags))
	rootCmd.AddCommand(NewDeleteCmd(globalFlags))
	rootCmd.AddCommand(NewSSHCmd(globalFlags))
	rootCmd.AddCommand(NewVersionCmd())
	rootCmd.AddCommand(NewStopCmd(globalFlags))
	rootCmd.AddCommand(NewListCmd(globalFlags))
	return rootCmd
}
