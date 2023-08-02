package container

import (
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/compress"
	config2 "github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/setup"
	"github.com/loft-sh/devpod/pkg/ide/fleet"
	"github.com/loft-sh/devpod/pkg/ide/jetbrains"
	"github.com/loft-sh/devpod/pkg/ide/jupyter"
	"github.com/loft-sh/devpod/pkg/ide/openvscode"
	"github.com/loft-sh/devpod/pkg/ide/vscode"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/single"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// SetupContainerCmd holds the cmd flags
type SetupContainerCmd struct {
	*flags.GlobalFlags

	ChownWorkspace bool
	WorkspaceInfo  string
	SetupInfo      string
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
	setupContainerCmd.Flags().BoolVar(&cmd.ChownWorkspace, "chown-workspace", false, "If DevPod should chown the workspace to the remote user")
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
	err = setup.SetupContainer(setupInfo, workspaceInfo, cmd.ChownWorkspace, log.Default)
	if err != nil {
		return err
	}

	// install IDE
	err = cmd.installIDE(setupInfo, workspaceInfo, log.Default)
	if err != nil {
		return err
	}

	// start container daemon if necessary
	if !workspaceInfo.CLIOptions.Proxy && !workspaceInfo.CLIOptions.DisableDaemon && workspaceInfo.Agent.ContainerTimeout != "" {
		err = single.Single("devpod.daemon.pid", func() (*exec.Cmd, error) {
			log.Default.Debugf("Start DevPod Container Daemon with Inactivity Timeout %s", workspaceInfo.Agent.ContainerTimeout)
			binaryPath, err := os.Executable()
			if err != nil {
				return nil, err
			}

			return exec.Command(binaryPath, "agent", "container", "daemon", "--timeout", workspaceInfo.Agent.ContainerTimeout), nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (cmd *SetupContainerCmd) installIDE(setupInfo *config.Result, workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error {
	switch workspaceInfo.Workspace.IDE.Name {
	case string(config2.IDENone):
		return nil
	case string(config2.IDEVSCode):
		return cmd.setupVSCode(setupInfo, workspaceInfo, log)
	case string(config2.IDEOpenVSCode):
		return cmd.setupOpenVSCode(setupInfo, workspaceInfo, log)
	case string(config2.IDEGoland):
		return jetbrains.NewGolandServer(config.GetRemoteUser(setupInfo), workspaceInfo.Workspace.IDE.Options, log).Install()
	case string(config2.IDEPyCharm):
		return jetbrains.NewPyCharmServer(config.GetRemoteUser(setupInfo), workspaceInfo.Workspace.IDE.Options, log).Install()
	case string(config2.IDEPhpStorm):
		return jetbrains.NewPhpStorm(config.GetRemoteUser(setupInfo), workspaceInfo.Workspace.IDE.Options, log).Install()
	case string(config2.IDEIntellij):
		return jetbrains.NewIntellij(config.GetRemoteUser(setupInfo), workspaceInfo.Workspace.IDE.Options, log).Install()
	case string(config2.IDECLion):
		return jetbrains.NewCLionServer(config.GetRemoteUser(setupInfo), workspaceInfo.Workspace.IDE.Options, log).Install()
	case string(config2.IDERider):
		return jetbrains.NewRiderServer(config.GetRemoteUser(setupInfo), workspaceInfo.Workspace.IDE.Options, log).Install()
	case string(config2.IDERubyMine):
		return jetbrains.NewRubyMineServer(config.GetRemoteUser(setupInfo), workspaceInfo.Workspace.IDE.Options, log).Install()
	case string(config2.IDEWebStorm):
		return jetbrains.NewWebStormServer(config.GetRemoteUser(setupInfo), workspaceInfo.Workspace.IDE.Options, log).Install()
	case string(config2.IDEFleet):
		return fleet.NewFleetServer(config.GetRemoteUser(setupInfo), workspaceInfo.Workspace.IDE.Options, log).Install(setupInfo.SubstitutionContext.ContainerWorkspaceFolder)
	case string(config2.IDEJupyterNotebook):
		return jupyter.NewJupyterNotebookServer(setupInfo.SubstitutionContext.ContainerWorkspaceFolder, config.GetRemoteUser(setupInfo), workspaceInfo.Workspace.IDE.Options, log).Install()
	}

	return nil
}

func (cmd *SetupContainerCmd) setupVSCode(setupInfo *config.Result, workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error {
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
	err := vscode.NewVSCodeServer(vsCodeConfiguration.Extensions, settings, user, workspaceInfo.Workspace.IDE.Options, log).Install()
	if err != nil {
		return err
	}

	// don't install code-server if we don't have settings or extensions
	if len(vsCodeConfiguration.Settings) == 0 && len(vsCodeConfiguration.Extensions) == 0 {
		return nil
	}

	if len(vsCodeConfiguration.Extensions) == 0 {
		return nil
	}

	return single.Single("vscode-async.pid", func() (*exec.Cmd, error) {
		log.Infof("Install extensions '%s' in the background", strings.Join(vsCodeConfiguration.Extensions, ","))
		binaryPath, err := os.Executable()
		if err != nil {
			return nil, err
		}

		return exec.Command(binaryPath, "agent", "container", "vscode-async", "--setup-info", cmd.SetupInfo), nil
	})
}

func setupVSCodeExtensions(setupInfo *config.Result, log log.Logger) error {
	vsCodeConfiguration := config.GetVSCodeConfiguration(setupInfo.MergedConfig)
	user := config.GetRemoteUser(setupInfo)
	return vscode.NewVSCodeServer(vsCodeConfiguration.Extensions, "", user, nil, log).InstallExtensions()
}

func setupOpenVSCodeExtensions(setupInfo *config.Result, log log.Logger) error {
	vsCodeConfiguration := config.GetVSCodeConfiguration(setupInfo.MergedConfig)
	user := config.GetRemoteUser(setupInfo)
	return openvscode.NewOpenVSCodeServer(vsCodeConfiguration.Extensions, "", user, "", "", nil, log).InstallExtensions()
}

func (cmd *SetupContainerCmd) setupOpenVSCode(setupInfo *config.Result, workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error {
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
	openVSCode := openvscode.NewOpenVSCodeServer(vsCodeConfiguration.Extensions, settings, user, "0.0.0.0", strconv.Itoa(openvscode.DefaultVSCodePort), workspaceInfo.Workspace.IDE.Options, log)

	// install open vscode
	err := openVSCode.Install()
	if err != nil {
		return err
	}

	// install extensions in background
	if len(vsCodeConfiguration.Extensions) > 0 {
		err = single.Single("openvscode-async.pid", func() (*exec.Cmd, error) {
			log.Infof("Install extensions '%s' in the background", strings.Join(vsCodeConfiguration.Extensions, ","))
			binaryPath, err := os.Executable()
			if err != nil {
				return nil, err
			}

			return exec.Command(binaryPath, "agent", "container", "openvscode-async", "--setup-info", cmd.SetupInfo), nil
		})
		if err != nil {
			return errors.Wrap(err, "install extensions")
		}
	}

	// start the server in the background
	return openVSCode.Start()
}
