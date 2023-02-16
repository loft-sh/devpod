package agent

import (
	"encoding/json"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/setup"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/vscode"
	"github.com/spf13/cobra"
	"os"
)

// SetupContainerCmd holds the cmd flags
type SetupContainerCmd struct {
	*flags.GlobalFlags

	WorkspaceInfo string
	SetupInfo     string
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
	setupContainerCmd.Flags().StringVar(&cmd.WorkspaceInfo, "workspace-info", "", "The workspace info")
	setupContainerCmd.Flags().StringVar(&cmd.SetupInfo, "setup-info", "", "The container setup info")
	_ = setupContainerCmd.MarkFlagRequired("setup-info")
	return setupContainerCmd
}

// Run runs the command logic
func (cmd *SetupContainerCmd) Run(_ *cobra.Command, _ []string) error {
	workspaceInfo, _, err := decodeWorkspaceInfo(cmd.WorkspaceInfo)
	if err != nil {
		return err
	}

	decompressed, err := compress.Decompress(cmd.SetupInfo)
	if err != nil {
		return err
	}

	setupInfo := &config.Result{}
	err = json.Unmarshal([]byte(decompressed), setupInfo)
	if err != nil {
		return err
	}

	// setting up container
	err = setup.SetupContainer(setupInfo)
	if err != nil {
		return err
	}

	// install IDE
	err = setupVSCode(setupInfo, workspaceInfo)
	if err != nil {
		return err
	}

	return nil
}

func setupVSCode(setupInfo *config.Result, workspaceInfo *provider2.AgentWorkspaceInfo) error {
	if workspaceInfo.Workspace.IDE.VSCode == nil {
		return nil
	}

	vsCodeConfiguration := config.GetVSCodeConfiguration(setupInfo.MergedConfig)
	settings := ""
	if len(vsCodeConfiguration.Settings) > 0 {
		out, err := json.Marshal(vsCodeConfiguration.Settings)
		if err != nil {
			return err
		}

		settings = string(out)
	}

	user := config.GetRemoteUser(setupInfo)
	if workspaceInfo.Workspace.IDE.VSCode.Browser {
		installer := &vscode.OpenVSCodeServer{}
		return installer.Install(vsCodeConfiguration.Extensions, settings, user, os.Stdout)
	}

	// don't install code-server if we don't have settings or extensions
	if len(vsCodeConfiguration.Settings) == 0 && len(vsCodeConfiguration.Extensions) == 0 {
		return nil
	}

	installer := &vscode.VSCodeServer{}
	return installer.Install(vsCodeConfiguration.Extensions, settings, user, os.Stdout)
}
