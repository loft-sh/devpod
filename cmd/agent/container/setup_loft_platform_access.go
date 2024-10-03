package container

import (
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/loftconfig"
	"github.com/loft-sh/log"

	"github.com/spf13/cobra"
)

type SetupLoftPlatformAccessCmd struct {
	*flags.GlobalFlags

	Context  string
	Provider string
	Port     int
}

// NewSetupLoftPlatformAccessCmd creates a new setup-loft-platform-access command
// This agent command can be used to inject loft platform configuration from local machine to workspace.
func NewSetupLoftPlatformAccessCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &SetupLoftPlatformAccessCmd{
		GlobalFlags: flags,
	}

	setupLoftPlatformAccessCmd := &cobra.Command{
		Use:   "setup-loft-platform-access",
		Short: "used to setup Loft Platform access",
		RunE:  cmd.Run,
	}

	setupLoftPlatformAccessCmd.Flags().StringVar(&cmd.Context, "context", "", "context to use")
	setupLoftPlatformAccessCmd.Flags().StringVar(&cmd.Provider, "provider", "", "provider to use")
	setupLoftPlatformAccessCmd.Flags().IntVar(&cmd.Port, "port", 0, "If specified, will use the given port")

	return setupLoftPlatformAccessCmd
}

// Run executes main command logic.
// It fetches Loft Platform credentials from credentials server and sets it up inside the workspace.
func (c *SetupLoftPlatformAccessCmd) Run(_ *cobra.Command, args []string) error {
	logger := log.Default.ErrorStreamOnly()

	loftConfig, err := loftconfig.GetLoftConfig(c.Context, c.Provider, c.Port, logger)
	if err != nil {
		return err
	}

	if loftConfig == nil {
		logger.Debug("Got empty loft config response, Loft Platform access won't be set up.")
		return nil
	}

	if err := loftconfig.AuthDevpodCliToPlatform(loftConfig, logger); err != nil {
		return err
	}

	if err := loftconfig.AuthVClusterCliToPlatform(loftConfig, logger); err != nil {
		return err
	}

	return nil
}
