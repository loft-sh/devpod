package cmd

import (
	"github.com/loft-sh/devpod/pkg/upgrade"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// UpgradeCmd is a struct that defines a command call for "upgrade"
type UpgradeCmd struct {
	log     log.Logger
	Version string
}

// NewUpgradeCmd creates a new upgrade command
func NewUpgradeCmd() *cobra.Command {
	cmd := &UpgradeCmd{log: log.GetInstance()}
	upgradeCmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade the DevPod CLI to the newest version",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}

	upgradeCmd.Flags().StringVar(&cmd.Version, "version", "", "The version to update to. Defaults to the latest stable version available")
	return upgradeCmd
}

// Run executes the command logic
func (cmd *UpgradeCmd) Run(*cobra.Command, []string) error {
	err := upgrade.Upgrade(cmd.Version, cmd.log)
	if err != nil {
		return errors.Errorf("unable to upgrade: %v", err)
	}

	return nil
}
