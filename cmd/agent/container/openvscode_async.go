package container

import (
	"encoding/json"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// OpenVSCodeAsyncCmd holds the cmd flags
type OpenVSCodeAsyncCmd struct {
	*flags.GlobalFlags

	SetupInfo string
}

// NewOpenVSCodeAsyncCmd creates a new command
func NewOpenVSCodeAsyncCmd() *cobra.Command {
	cmd := &OpenVSCodeAsyncCmd{}
	vsCodeAsyncCmd := &cobra.Command{
		Use:   "openvscode-async",
		Short: "Starts openvscode",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}
	vsCodeAsyncCmd.Flags().StringVar(&cmd.SetupInfo, "setup-info", "", "The container setup info")
	_ = vsCodeAsyncCmd.MarkFlagRequired("setup-info")
	return vsCodeAsyncCmd
}

// Run runs the command logic
func (cmd *OpenVSCodeAsyncCmd) Run(_ *cobra.Command, _ []string) error {
	log.Default.Debugf("Start setting up container...")
	decompressed, err := compress.Decompress(cmd.SetupInfo)
	if err != nil {
		return err
	}

	setupInfo := &config.Result{}
	err = json.Unmarshal([]byte(decompressed), setupInfo)
	if err != nil {
		return err
	}

	// install IDE
	return setupOpenVSCodeExtensions(setupInfo, log.Default)
}
