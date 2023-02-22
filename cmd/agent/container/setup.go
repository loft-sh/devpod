package container

import (
	"encoding/json"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/setup"
	"github.com/loft-sh/devpod/pkg/ide/openvscode"
	"github.com/loft-sh/devpod/pkg/ide/vscode"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/single"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"strconv"
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
		Use:   "setup",
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
	log.Default.Debugf("Start setting up container...")
	workspaceInfo, _, err := agent.DecodeWorkspaceInfo(cmd.WorkspaceInfo)
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
	err = setup.SetupContainer(setupInfo, log.Default)
	if err != nil {
		return err
	}

	// install IDE
	err = installIDE(setupInfo, workspaceInfo, log.Default)
	if err != nil {
		return err
	}

	// start container daemon if necessary
	if workspaceInfo.Workspace.Provider.Mode == provider2.ModeSingle && workspaceInfo.Workspace.Provider.Agent.Timeout != "" {
		err = single.Single("devpod.daemon.pid", func() (*exec.Cmd, error) {
			log.Default.Debugf("Start DevPod Container Daemon with Inactivity Timeout %s", workspaceInfo.Workspace.Provider.Agent.Timeout)
			binaryPath, err := os.Executable()
			if err != nil {
				return nil, err
			}

			return exec.Command(binaryPath, "agent", "container", "daemon", "--timeout", workspaceInfo.Workspace.Provider.Agent.Timeout), nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func installIDE(setupInfo *config.Result, workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error {
	switch workspaceInfo.Workspace.IDE.IDE {
	case provider2.IDENone:
		return nil
	case provider2.IDEVSCode:
		return setupVSCode(setupInfo, log)
	case provider2.IDEOpenVSCode:
		return setupOpenVSCode(setupInfo, log)
	}

	return nil
}

func setupVSCode(setupInfo *config.Result, log log.Logger) error {
	log.Debugf("Setup vscode...")
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

	// don't install code-server if we don't have settings or extensions
	if len(vsCodeConfiguration.Settings) == 0 && len(vsCodeConfiguration.Extensions) == 0 {
		return nil
	}

	return vscode.NewVSCodeServer(vsCodeConfiguration.Extensions, settings, user, log).Install()
}

func setupOpenVSCode(setupInfo *config.Result, log log.Logger) error {
	log.Debugf("Setup openvscode...")
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
	return openvscode.NewOpenVSCodeServer(vsCodeConfiguration.Extensions, settings, user, "0.0.0.0", strconv.Itoa(openvscode.DefaultVSCodePort), log).Install()
}
