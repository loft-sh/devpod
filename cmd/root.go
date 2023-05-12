package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/loft-sh/devpod/cmd/agent"
	"github.com/loft-sh/devpod/cmd/context"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/cmd/helper"
	"github.com/loft-sh/devpod/cmd/ide"
	"github.com/loft-sh/devpod/cmd/machine"
	"github.com/loft-sh/devpod/cmd/provider"
	"github.com/loft-sh/devpod/cmd/use"
	log2 "github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/telemetry"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var (
	globalFlags *flags.GlobalFlags
)

// NewRootCmd returns a new root command
func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "devpod",
		Short:         "DevPod",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cobraCmd *cobra.Command, args []string) error {
			telemetry.Collector.SetCLIData(cobraCmd, globalFlags)

			if globalFlags.LogOutput == "json" {
				log2.Default.SetFormat(log2.JSONFormat)
			} else if globalFlags.LogOutput != "plain" {
				return fmt.Errorf("unrecognized log format %s, needs to be either plain or json", globalFlags.LogOutput)
			}

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
	defer func() {
		// recover from panic in order to log it via telemetry
		if err := recover(); err != nil {
			telemetry.Collector.RecordCMDFinishedEvent(fmt.Errorf("panic: %v", err))
			log2.Default.Fatal(fmt.Errorf("panic: %v", err))
		}
	}()

	// build the root command
	rootCmd := BuildRoot()

	// execute command
	startTime := time.Now()
	err := rootCmd.Execute()
	if err != nil || time.Since(startTime) > telemetry.TelemetrySendFinishedAfter {
		telemetry.Collector.RecordCMDFinishedEvent(err)
	}

	// ensure that CMDStart telemetry async upload has finished if it started
	if telemetry.CMDStartedDoneChan != nil {
		<-(*telemetry.CMDStartedDoneChan)
	}

	if err != nil {
		//nolint:all
		if sshExitErr, ok := err.(*ssh.ExitError); ok {
			os.Exit(sshExitErr.ExitStatus())
		}

		//nolint:all
		if execExitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(execExitErr.ExitCode())
		}

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
	rootCmd.AddCommand(use.NewUseCmd(globalFlags))
	rootCmd.AddCommand(helper.NewHelperCmd(globalFlags))
	rootCmd.AddCommand(ide.NewIDECmd(globalFlags))
	rootCmd.AddCommand(machine.NewMachineCmd(globalFlags))
	rootCmd.AddCommand(context.NewContextCmd(globalFlags))
	rootCmd.AddCommand(NewUpCmd(globalFlags))
	rootCmd.AddCommand(NewDeleteCmd(globalFlags))
	rootCmd.AddCommand(NewSSHCmd(globalFlags))
	rootCmd.AddCommand(NewVersionCmd())
	rootCmd.AddCommand(NewStopCmd(globalFlags))
	rootCmd.AddCommand(NewListCmd(globalFlags))
	rootCmd.AddCommand(NewStatusCmd(globalFlags))
	rootCmd.AddCommand(NewBuildCmd(globalFlags))
	rootCmd.AddCommand(NewLogsDaemonCmd(globalFlags))
	return rootCmd
}
