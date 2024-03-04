package container

import (
	"encoding/json"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/ide/vscode"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// VSCodeAsyncCmd holds the cmd flags
type VSCodeAsyncCmd struct {
	*flags.GlobalFlags

	SetupInfo      string
	ReleaseChannel string
}

// NewVSCodeAsyncCmd creates a new command
func NewVSCodeAsyncCmd() *cobra.Command {
	cmd := &VSCodeAsyncCmd{}
	vsCodeAsyncCmd := &cobra.Command{
		Use:   "vscode-async",
		Short: "Starts vscode",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}
	vsCodeAsyncCmd.Flags().StringVar(&cmd.SetupInfo, "setup-info", "", "The container setup info")
	vsCodeAsyncCmd.Flags().StringVar(&cmd.ReleaseChannel, "release-channel", string(vscode.ReleaseChannelStable), "The release channel to use for vscode")
	_ = vsCodeAsyncCmd.MarkFlagRequired("setup-info")
	return vsCodeAsyncCmd
}

// Run runs the command logic
func (cmd *VSCodeAsyncCmd) Run(_ *cobra.Command, _ []string) error {
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
	err = setupVSCodeExtensions(setupInfo, vscode.ReleaseChannel(cmd.ReleaseChannel), log.Default)
	if err != nil {
		return err
	}

	return nil
}
