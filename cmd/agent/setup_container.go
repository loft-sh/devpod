package agent

import (
	"encoding/json"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/setup"
	"github.com/spf13/cobra"
)

// SetupContainerCmd holds the cmd flags
type SetupContainerCmd struct {
	*flags.GlobalFlags

	SetupInfo string
}

// NewSetupContainerCmd creates a new command
func NewSetupContainerCmd() *cobra.Command {
	cmd := &SetupContainerCmd{}
	setupContainerCmd := &cobra.Command{
		Use:   "setup-container",
		Short: "Sets up a container",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}
	setupContainerCmd.Flags().StringVar(&cmd.SetupInfo, "setup-info", "", "The container setup info")
	_ = setupContainerCmd.MarkFlagRequired("setup-info")
	return setupContainerCmd
}

// Run runs the command logic
func (cmd *SetupContainerCmd) Run(_ *cobra.Command, _ []string) error {
	decompressed, err := compress.Decompress(cmd.SetupInfo)
	if err != nil {
		return err
	}

	setupInfo := &config.Result{}
	err = json.Unmarshal([]byte(decompressed), setupInfo)
	if err != nil {
		return err
	}

	err = setup.SetupContainer(setupInfo)
	if err != nil {
		return err
	}

	return nil
}
