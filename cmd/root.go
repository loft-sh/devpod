package cmd

import (
	"fmt"
	"github.com/loft-sh/devpod/cmd/agent"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/cmd/provider"
	"os"

	"github.com/spf13/cobra"
)

// NewRootCmd returns a new root command
func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "devpod",
		Short:         "DevPod",
		SilenceUsage:  true,
		SilenceErrors: true,
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
		_, _ = fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}

// BuildRoot creates a new root command from the
func BuildRoot() *cobra.Command {
	rootCmd := NewRootCmd()
	persistentFlags := rootCmd.PersistentFlags()
	globalFlags := flags.SetGlobalFlags(persistentFlags)

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
