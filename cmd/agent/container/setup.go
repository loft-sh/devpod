package container

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/agent/tunnelserver"
	"github.com/loft-sh/devpod/pkg/compress"
	config2 "github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/copy"
	"github.com/loft-sh/devpod/pkg/credentials"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/setup"
	"github.com/loft-sh/devpod/pkg/dockercredentials"
	"github.com/loft-sh/devpod/pkg/envfile"
	"github.com/loft-sh/devpod/pkg/extract"
	"github.com/loft-sh/devpod/pkg/ide/fleet"
	"github.com/loft-sh/devpod/pkg/ide/jetbrains"
	"github.com/loft-sh/devpod/pkg/ide/jupyter"
	"github.com/loft-sh/devpod/pkg/ide/openvscode"
	"github.com/loft-sh/devpod/pkg/ide/vscode"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/single"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var DockerlessImageConfigOutput = "/.dockerless/image.json"

// SetupContainerCmd holds the cmd flags
type SetupContainerCmd struct {
	*flags.GlobalFlags

	ChownWorkspace bool
	StreamMounts   bool
	WorkspaceInfo  string
	SetupInfo      string
}

// NewSetupContainerCmd creates a new command
func NewSetupContainerCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &SetupContainerCmd{
		GlobalFlags: flags,
	}
	setupContainerCmd := &cobra.Command{
		Use:   "setup",
		Short: "Sets up a container",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background())
		},
	}
	setupContainerCmd.Flags().BoolVar(&cmd.StreamMounts, "stream-mounts", false, "If true, will try to stream the bind mounts from the host")
	setupContainerCmd.Flags().BoolVar(&cmd.ChownWorkspace, "chown-workspace", false, "If DevPod should chown the workspace to the remote user")
	setupContainerCmd.Flags().StringVar(&cmd.WorkspaceInfo, "workspace-info", "", "The workspace info")
	setupContainerCmd.Flags().StringVar(&cmd.SetupInfo, "setup-info", "", "The container setup info")
	_ = setupContainerCmd.MarkFlagRequired("setup-info")
	return setupContainerCmd
}

// Run runs the command logic
func (cmd *SetupContainerCmd) Run(ctx context.Context) error {
	// create a grpc client
	tunnelClient, err := tunnelserver.NewTunnelClient(os.Stdin, os.Stdout, true)
	if err != nil {
		return fmt.Errorf("error creating tunnel client: %w", err)
	}

	// create debug logger
	log := tunnelserver.NewTunnelLogger(ctx, tunnelClient, cmd.Debug)
	log.Debugf("Created logger")

	// this message serves as a ping to the client
	_, err = tunnelClient.Ping(ctx, &tunnel.Empty{})
	if err != nil {
		return errors.Wrap(err, "ping client")
	}

	// start setting up container
	log.Debugf("Start setting up container...")
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

	// sync mounts
	if cmd.StreamMounts {
		mounts := config.GetMounts(setupInfo)
		for _, m := range mounts {
			files, err := os.ReadDir(m.Target)
			if err == nil && len(files) > 0 {
				continue
			}

			// stream mount
			log.Infof("Copy %s into DevContainer %s", m.Source, m.Target)
			stream, err := tunnelClient.StreamMount(ctx, &tunnel.StreamMountRequest{Mount: m.String()})
			if err != nil {
				return fmt.Errorf("init stream mount %s: %w", m.String(), err)
			}

			// target folder
			err = extract.Extract(tunnelserver.NewStreamReader(stream, log), m.Target)
			if err != nil {
				return fmt.Errorf("stream stream mount %s: %w", m.String(), err)
			}
		}
	}

	// do dockerless build
	err = dockerlessBuild(ctx, workspaceInfo, tunnelClient, log)
	if err != nil {
		return fmt.Errorf("dockerless build: %w", err)
	}

	// fill container env
	err = fillContainerEnv(setupInfo)
	if err != nil {
		return err
	}

	// setup container
	err = setup.SetupContainer(setupInfo, workspaceInfo, cmd.ChownWorkspace, log)
	if err != nil {
		return err
	}

	// install IDE
	err = cmd.installIDE(setupInfo, workspaceInfo, log)
	if err != nil {
		return err
	}

	// start container daemon if necessary
	if !workspaceInfo.CLIOptions.Proxy && !workspaceInfo.CLIOptions.DisableDaemon && workspaceInfo.Agent.ContainerTimeout != "" {
		err = single.Single("devpod.daemon.pid", func() (*exec.Cmd, error) {
			log.Debugf("Start DevPod Container Daemon with Inactivity Timeout %s", workspaceInfo.Agent.ContainerTimeout)
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

	out, err := json.Marshal(setupInfo)
	if err != nil {
		return fmt.Errorf("marshal setup info: %w", err)
	}

	_, err = tunnelClient.SendResult(ctx, &tunnel.Message{Message: string(out)})
	if err != nil {
		return fmt.Errorf("send result: %w", err)
	}

	return nil
}

func fillContainerEnv(setupInfo *config.Result) error {
	// set remote-env
	if setupInfo.MergedConfig.RemoteEnv == nil {
		setupInfo.MergedConfig.RemoteEnv = make(map[string]string)
	}
	if _, ok := setupInfo.MergedConfig.RemoteEnv["PATH"]; !ok {
		setupInfo.MergedConfig.RemoteEnv["PATH"] = "${containerEnv:PATH}"
	}

	// merge config
	newMergedConfig := &config.MergedDevContainerConfig{}
	err := config.SubstituteContainerEnv(config.ListToObject(os.Environ()), setupInfo.MergedConfig, newMergedConfig)
	if err != nil {
		return errors.Wrap(err, "substitute container env")
	}
	setupInfo.MergedConfig = newMergedConfig
	return nil
}

func dockerlessBuild(ctx context.Context, workspaceInfo *provider2.AgentWorkspaceInfo, client tunnel.TunnelClient, log log.Logger) error {
	if os.Getenv("DOCKERLESS") != "true" {
		return nil
	}

	_, err := os.Stat(DockerlessImageConfigOutput)
	if err == nil {
		log.Debugf("Skip dockerless build, because container was built already")
		return nil
	}

	buildContext := os.Getenv("DOCKERLESS_CONTEXT")
	if buildContext == "" {
		log.Debugf("Build context is missing for dockerless build")
		return nil
	}

	// check if build info is there
	fallbackDir := filepath.Join(config.DevPodDockerlessBuildInfoFolder, config.DevPodContextFeatureFolder)
	buildInfoDir := filepath.Join(buildContext, config.DevPodContextFeatureFolder)
	_, err = os.Stat(buildInfoDir)
	if err != nil {
		// try to rename from fallback dir
		err = copy.RenameDirectory(fallbackDir, buildInfoDir)
		if err != nil {
			return fmt.Errorf("rename dir: %w", err)
		}

		_, err = os.Stat(buildInfoDir)
		if err != nil {
			return fmt.Errorf("couldn't find build dir %s: %w", buildInfoDir, err)
		}
	}

	binaryPath, err := os.Executable()
	if err != nil {
		return err
	}

	// configure credentials
	if workspaceInfo.Agent.InjectDockerCredentials != "false" {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()

		// configure the docker credentials
		dockerCredentialsDir, err := configureDockerCredentials(ctx, cancel, client, log)
		if err != nil {
			log.Errorf("Error configuring docker credentials: %v", err)
		} else {
			defer func() {
				_ = os.Unsetenv("DOCKER_CONFIG")
				_ = os.RemoveAll(dockerCredentialsDir)
			}()
		}
	}

	// build args
	args := []string{"build", "--ignore-path", binaryPath}
	args = append(args, parseIgnorePaths(workspaceInfo.Agent.DockerlessIgnorePaths)...)

	// write output to log
	writer := log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// start building
	log.Infof("Start dockerless building with kaniko...")
	cmd := exec.CommandContext(ctx, "/.dockerless/dockerless", args...)
	cmd.Stdout = writer
	cmd.Stderr = writer
	cmd.Env = os.Environ()
	err = cmd.Run()
	if err != nil {
		return err
	}

	// add container env to envfile.json
	rawConfig, err := os.ReadFile(DockerlessImageConfigOutput)
	if err != nil {
		return err
	}

	// parse config file
	configFile := &v1.ConfigFile{}
	err = json.Unmarshal(rawConfig, configFile)
	if err != nil {
		return fmt.Errorf("parse container config: %w", err)
	}

	// apply env
	envfile.MergeAndApply(config.ListToObject(configFile.Config.Env), log)

	// rename build path
	_ = os.RemoveAll(fallbackDir)
	err = copy.RenameDirectory(buildInfoDir, fallbackDir)
	if err != nil {
		log.Debugf("Error renaming dir %s: %v", buildInfoDir, err)
		return nil
	}

	return nil
}

func parseIgnorePaths(ignorePaths string) []string {
	retPaths := []string{}
	splitted := strings.Split(ignorePaths, ",")
	for _, s := range splitted {
		retPaths = append(retPaths, "--ignore-path", strings.TrimSpace(s))
	}

	return retPaths
}

func configureDockerCredentials(
	ctx context.Context,
	cancel context.CancelFunc,
	client tunnel.TunnelClient,
	log log.Logger,
) (string, error) {
	serverPort, err := credentials.StartCredentialsServer(ctx, cancel, client, log)
	if err != nil {
		return "", err
	}

	dockerCredentials, err := dockercredentials.ConfigureCredentialsDockerless("/.dockerless/.docker", serverPort, log)
	if err != nil {
		return "", err
	}

	return dockerCredentials, nil
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
