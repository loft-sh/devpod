package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/blang/semver"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/agent/tunnelserver"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/credentials"
	config2 "github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/sshtunnel"
	dpFlags "github.com/loft-sh/devpod/pkg/flags"
	"github.com/loft-sh/devpod/pkg/ide/fleet"
	"github.com/loft-sh/devpod/pkg/ide/jetbrains"
	"github.com/loft-sh/devpod/pkg/ide/jupyter"
	"github.com/loft-sh/devpod/pkg/ide/marimo"
	"github.com/loft-sh/devpod/pkg/ide/openvscode"
	"github.com/loft-sh/devpod/pkg/ide/vscode"
	"github.com/loft-sh/devpod/pkg/loft"
	open2 "github.com/loft-sh/devpod/pkg/open"
	"github.com/loft-sh/devpod/pkg/port"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/tunnel"
	"github.com/loft-sh/devpod/pkg/version"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

// UpCmd holds the up cmd flags
type UpCmd struct {
	provider2.CLIOptions
	*flags.GlobalFlags

	Machine string

	ProviderOptions []string

	ConfigureSSH            bool
	GPGAgentForwarding      bool
	OpenIDE                 bool
	SetupLoftPlatformAccess bool

	SSHConfigPath string

	DotfilesSource string
	DotfilesScript string
}

// NewUpCmd creates a new up command
func NewUpCmd(f *flags.GlobalFlags) *cobra.Command {
	cmd := &UpCmd{
		GlobalFlags: f,
	}
	upCmd := &cobra.Command{
		Use:   "up [flags] [workspace-path|workspace-name]",
		Short: "Starts a new workspace",
		PreRunE: func(_ *cobra.Command, args []string) error {
			absExtraDevContainerPaths := []string{}
			for _, extraPath := range cmd.ExtraDevContainerPaths {
				absExtraPath, err := filepath.Abs(extraPath)
				if err != nil {
					return err
				}

				absExtraDevContainerPaths = append(absExtraDevContainerPaths, absExtraPath)
			}
			cmd.ExtraDevContainerPaths = absExtraDevContainerPaths
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
			if err != nil {
				return err
			}

			// try to parse flags from env
			if err := mergeDevPodUpOptions(&cmd.CLIOptions); err != nil {
				return err
			}

			var logger log.Logger = log.Default
			if cmd.Proxy {
				logger = logger.ErrorStreamOnly()
				logger.Debug("Running in proxy mode")
				logger.Debug("Using error output stream")

				// merge context options from env
				config.MergeContextOptions(devPodConfig.Current(), os.Environ())
			}

			err = mergeEnvFromFiles(&cmd.CLIOptions)
			if err != nil {
				return err
			}

			var source *provider2.WorkspaceSource
			if cmd.Source != "" {
				source = provider2.ParseWorkspaceSource(cmd.Source)
				if source == nil {
					return fmt.Errorf("workspace source is missing")
				}
			}

			if cmd.SSHConfigPath == "" {
				cmd.SSHConfigPath = devPodConfig.ContextOption(config.ContextOptionSSHConfigPath)
			}

			ctx := context.Background()
			client, err := workspace2.ResolveWorkspace(
				ctx,
				devPodConfig,
				cmd.IDE,
				cmd.IDEOptions,
				args,
				cmd.ID,
				cmd.Machine,
				cmd.ProviderOptions,
				cmd.DevContainerImage,
				cmd.DevContainerPath,
				cmd.SSHConfigPath,
				source,
				cmd.UID,
				true,
				logger,
			)
			if err != nil {
				return err
			}

			if !cmd.Proxy {
				proInstance := getProInstance(devPodConfig, client.Provider(), logger)
				if proInstance != nil {
					cmd.SetupLoftPlatformAccess = true
				}

				err = checkProviderUpdate(devPodConfig, proInstance, logger)
				if err != nil {
					return err
				}
			}

			return cmd.Run(ctx, devPodConfig, client, logger)
		},
	}
	dpFlags.SetGitCredentialsFlags(upCmd.Flags(), &cmd.GitCredentialsFlags)
	upCmd.Flags().BoolVar(&cmd.ConfigureSSH, "configure-ssh", true, "If true will configure the ssh config to include the DevPod workspace")
	upCmd.Flags().BoolVar(&cmd.GPGAgentForwarding, "gpg-agent-forwarding", false, "If true forward the local gpg-agent to the DevPod workspace")
	upCmd.Flags().StringVar(&cmd.SSHConfigPath, "ssh-config", "", "The path to the ssh config to modify, if empty will use ~/.ssh/config")
	upCmd.Flags().StringVar(&cmd.DotfilesSource, "dotfiles", "", "The path or url to the dotfiles to use in the container")
	upCmd.Flags().StringVar(&cmd.DotfilesScript, "dotfiles-script", "", "The path in dotfiles directory to use to install the dotfiles, if empty will try to guess")
	upCmd.Flags().StringArrayVar(&cmd.IDEOptions, "ide-option", []string{}, "IDE option in the form KEY=VALUE")
	upCmd.Flags().StringVar(&cmd.DevContainerImage, "devcontainer-image", "", "The container image to use, this will override the devcontainer.json value in the project")
	upCmd.Flags().StringVar(&cmd.DevContainerPath, "devcontainer-path", "", "The path to the devcontainer.json relative to the project")
	upCmd.Flags().StringVar(&cmd.DevContainerSource, "devcontainer-source", "", "External devcontainer.json source")
	upCmd.Flags().StringArrayVar(&cmd.ExtraDevContainerPaths, "extra-devcontainer-path", []string{}, "The path to additional devcontainer.json files to override original devcontainer.json")
	upCmd.Flags().StringVar(&cmd.EnvironmentTemplate, "environment-template", "", "Environment template to use")
	_ = upCmd.Flags().MarkHidden("environment-template")
	upCmd.Flags().StringArrayVar(&cmd.ProviderOptions, "provider-option", []string{}, "Provider option in the form KEY=VALUE")
	upCmd.Flags().BoolVar(&cmd.Recreate, "recreate", false, "If true will remove any existing containers and recreate them")
	upCmd.Flags().BoolVar(&cmd.Reset, "reset", false, "If true will remove any existing containers including sources, and recreate them")
	upCmd.Flags().StringSliceVar(&cmd.PrebuildRepositories, "prebuild-repository", []string{}, "Docker repository that hosts devpod prebuilds for this workspace")
	upCmd.Flags().StringArrayVar(&cmd.WorkspaceEnv, "workspace-env", []string{}, "Extra env variables to put into the workspace. E.g. MY_ENV_VAR=MY_VALUE")
	upCmd.Flags().StringSliceVar(&cmd.WorkspaceEnvFile, "workspace-env-file", []string{}, "The path to files containing a list of extra env variables to put into the workspace. E.g. MY_ENV_VAR=MY_VALUE")
	upCmd.Flags().StringArrayVar(&cmd.InitEnv, "init-env", []string{}, "Extra env variables to inject during the initialization of the workspace. E.g. MY_ENV_VAR=MY_VALUE")
	upCmd.Flags().StringVar(&cmd.ID, "id", "", "The id to use for the workspace")
	upCmd.Flags().StringVar(&cmd.Machine, "machine", "", "The machine to use for this workspace. The machine needs to exist beforehand or the command will fail. If the workspace already exists, this option has no effect")
	upCmd.Flags().StringVar(&cmd.IDE, "ide", "", "The IDE to open the workspace in. If empty will use vscode locally or in browser")
	upCmd.Flags().BoolVar(&cmd.OpenIDE, "open-ide", true, "If this is false and an IDE is configured, DevPod will only install the IDE server backend, but not open it")
	upCmd.Flags().Var(&cmd.GitCloneStrategy, "git-clone-strategy", "The git clone strategy DevPod uses to checkout git based workspaces. Can be full (default), blobless, treeless or shallow")
	upCmd.Flags().StringVar(&cmd.GitSSHSigningKey, "git-ssh-signing-key", "", "The ssh key to use when signing git commits. Used to explicitly setup DevPod's ssh signature forwarding with given key. Should be same format as value of `git config user.signingkey`")
	upCmd.Flags().StringVar(&cmd.FallbackImage, "fallback-image", "", "The fallback image to use if no devcontainer configuration has been detected")

	upCmd.Flags().BoolVar(&cmd.DisableDaemon, "disable-daemon", false, "If enabled, will not install a daemon into the target machine to track activity")
	upCmd.Flags().StringVar(&cmd.Source, "source", "", "Optional source for the workspace. E.g. git:https://github.com/my-org/my-repo")
	upCmd.Flags().BoolVar(&cmd.Proxy, "proxy", false, "If true will forward agent requests to stdio")
	upCmd.Flags().BoolVar(&cmd.ForceCredentials, "force-credentials", false, "If true will always use local credentials")
	_ = upCmd.Flags().MarkHidden("force-credentials")
	upCmd.Flags().BoolVar(&cmd.SetupLoftPlatformAccess, "setup-loft-platform-access", false, "If true will setup Loft Platform access based on local configuration")
	_ = upCmd.Flags().MarkHidden("setup-loft-platform-access")

	upCmd.Flags().StringVar(&cmd.SSHKey, "ssh-key", "", "The ssh-key to use")
	_ = upCmd.Flags().MarkHidden("ssh-key")

	// testing
	upCmd.Flags().StringVar(&cmd.DaemonInterval, "daemon-interval", "", "TESTING ONLY")
	_ = upCmd.Flags().MarkHidden("daemon-interval")
	upCmd.Flags().BoolVar(&cmd.ForceDockerless, "force-dockerless", false, "TESTING ONLY")
	_ = upCmd.Flags().MarkHidden("force-dockerless")
	return upCmd
}

// Run runs the command logic
func (cmd *UpCmd) Run(
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.BaseWorkspaceClient,
	log log.Logger,
) error {
	// a reset implies a recreate
	if cmd.Reset {
		cmd.Recreate = true
	}

	// run devpod agent up
	result, err := cmd.devPodUp(ctx, devPodConfig, client, log)
	if err != nil {
		return err
	} else if result == nil {
		return fmt.Errorf("didn't receive a result back from agent")
	} else if cmd.Proxy {
		return nil
	}

	// get user from result
	user := config2.GetRemoteUser(result)

	var workdir string
	if result.MergedConfig != nil && result.MergedConfig.WorkspaceFolder != "" {
		workdir = result.MergedConfig.WorkspaceFolder
	}

	if client.WorkspaceConfig().Source.GitSubPath != "" {
		result.SubstitutionContext.ContainerWorkspaceFolder = filepath.Join(result.SubstitutionContext.ContainerWorkspaceFolder, client.WorkspaceConfig().Source.GitSubPath)
		workdir = result.SubstitutionContext.ContainerWorkspaceFolder
	}

	// configure container ssh
	if cmd.ConfigureSSH {
		devPodHome := ""
		envDevPodHome, ok := os.LookupEnv("DEVPOD_HOME")
		if ok {
			devPodHome = envDevPodHome
		}
		setupGPGAgentForwarding := cmd.GPGAgentForwarding || devPodConfig.ContextOption(config.ContextOptionGPGAgentForwarding) == "true"

		err = configureSSH(devPodConfig, client, cmd.SSHConfigPath, user, workdir, setupGPGAgentForwarding, devPodHome)
		if err != nil {
			return err
		}

		log.Infof("Run 'ssh %s.devpod' to ssh into the devcontainer", client.Workspace())
	}

	// setup git ssh signature
	if cmd.GitSSHSigningKey != "" {
		err = setupGitSSHSignature(cmd.GitSSHSigningKey, client, log)
		if err != nil {
			return err
		}
	}

	// setup loft platform access
	context := devPodConfig.Current()
	if cmd.SetupLoftPlatformAccess {
		err = setupLoftPlatformAccess(devPodConfig.DefaultContext, context.DefaultProvider, user, client, log)
		if err != nil {
			return err
		}
	}

	// setup dotfiles in the container
	err = setupDotfiles(cmd.DotfilesSource, cmd.DotfilesScript, client, devPodConfig, log)
	if err != nil {
		return err
	}

	// open ide
	if cmd.OpenIDE {
		ideConfig := client.WorkspaceConfig().IDE
		switch ideConfig.Name {
		case string(config.IDEVSCode):
			return vscode.Open(
				ctx,
				client.Workspace(),
				result.SubstitutionContext.ContainerWorkspaceFolder,
				vscode.Options.GetValue(ideConfig.Options, vscode.OpenNewWindow) == "true",
				vscode.FlavorStable,
				log,
			)
		case string(config.IDEVSCodeInsiders):
			return vscode.Open(
				ctx,
				client.Workspace(),
				result.SubstitutionContext.ContainerWorkspaceFolder,
				vscode.Options.GetValue(ideConfig.Options, vscode.OpenNewWindow) == "true",
				vscode.FlavorInsiders,
				log,
			)
		case string(config.IDECursor):
			return vscode.Open(
				ctx,
				client.Workspace(),
				result.SubstitutionContext.ContainerWorkspaceFolder,
				vscode.Options.GetValue(ideConfig.Options, vscode.OpenNewWindow) == "true",
				vscode.FlavorCursor,
				log,
			)
		case string(config.IDEPositron):
			return vscode.Open(
				ctx,
				client.Workspace(),
				result.SubstitutionContext.ContainerWorkspaceFolder,
				vscode.Options.GetValue(ideConfig.Options, vscode.OpenNewWindow) == "true",
				vscode.FlavorPositron,
				log,
			)
		case string(config.IDEOpenVSCode):
			return startVSCodeInBrowser(
				cmd.GPGAgentForwarding,
				ctx,
				devPodConfig,
				client,
				result.SubstitutionContext.ContainerWorkspaceFolder,
				user,
				ideConfig.Options,
				cmd.GitUsername,
				cmd.GitToken,
				log,
			)
		case string(config.IDERustRover):
			return jetbrains.NewRustRoverServer(config2.GetRemoteUser(result), ideConfig.Options, log).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
		case string(config.IDEGoland):
			return jetbrains.NewGolandServer(config2.GetRemoteUser(result), ideConfig.Options, log).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
		case string(config.IDEPyCharm):
			return jetbrains.NewPyCharmServer(config2.GetRemoteUser(result), ideConfig.Options, log).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
		case string(config.IDEPhpStorm):
			return jetbrains.NewPhpStorm(config2.GetRemoteUser(result), ideConfig.Options, log).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
		case string(config.IDEIntellij):
			return jetbrains.NewIntellij(config2.GetRemoteUser(result), ideConfig.Options, log).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
		case string(config.IDECLion):
			return jetbrains.NewCLionServer(config2.GetRemoteUser(result), ideConfig.Options, log).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
		case string(config.IDERider):
			return jetbrains.NewRiderServer(config2.GetRemoteUser(result), ideConfig.Options, log).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
		case string(config.IDERubyMine):
			return jetbrains.NewRubyMineServer(config2.GetRemoteUser(result), ideConfig.Options, log).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
		case string(config.IDEWebStorm):
			return jetbrains.NewWebStormServer(config2.GetRemoteUser(result), ideConfig.Options, log).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
		case string(config.IDEFleet):
			return startFleet(ctx, client, log)
		case string(config.IDEJupyterNotebook):
			return startJupyterNotebookInBrowser(
				cmd.GPGAgentForwarding,
				ctx,
				devPodConfig,
				client,
				user,
				ideConfig.Options,
				cmd.GitUsername,
				cmd.GitToken,
				log,
			)
		case string(config.IDEJupyterDesktop):
			return startJupyterDesktop(
				cmd.GPGAgentForwarding,
				ctx,
				devPodConfig,
				client,
				user,
				ideConfig.Options,
				cmd.GitUsername,
				cmd.GitToken,
				log)
		case string(config.IDEMarimo):
			return startMarimoInBrowser(
				cmd.GPGAgentForwarding,
				ctx,
				devPodConfig,
				client,
				user,
				ideConfig.Options,
				cmd.GitUsername,
				cmd.GitToken,
				log)
		}
	}

	return nil
}

func (cmd *UpCmd) devPodUp(
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.BaseWorkspaceClient,
	log log.Logger,
) (*config2.Result, error) {
	err := client.Lock(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Unlock()

	// get result
	var result *config2.Result

	switch client := client.(type) {
	case client2.WorkspaceClient:
		result, err = cmd.devPodUpMachine(ctx, devPodConfig, client, log)
		if err != nil {
			return nil, err
		}
	case client2.ProxyClient:
		result, err = cmd.devPodUpProxy(ctx, client, log)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported client type: %T", client)
	}

	// save result to file
	err = provider2.SaveWorkspaceResult(client.WorkspaceConfig(), result)
	if err != nil {
		return nil, fmt.Errorf("save workspace result: %w", err)
	}

	return result, nil
}

func (cmd *UpCmd) devPodUpProxy(
	ctx context.Context,
	client client2.ProxyClient,
	log log.Logger,
) (*config2.Result, error) {
	// create pipes
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer stdoutWriter.Close()
	defer stdinWriter.Close()

	// start machine on stdio
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// create up command
	errChan := make(chan error, 1)
	go func() {
		defer log.Debugf("Done executing up command")
		defer cancel()

		// build devpod up options
		workspace := client.WorkspaceConfig()
		baseOptions := cmd.CLIOptions
		baseOptions.ID = workspace.ID
		baseOptions.DevContainerPath = workspace.DevContainerPath
		baseOptions.DevContainerImage = workspace.DevContainerImage
		baseOptions.IDE = workspace.IDE.Name
		baseOptions.IDEOptions = nil
		baseOptions.Source = workspace.Source.String()
		for optionName, optionValue := range workspace.IDE.Options {
			baseOptions.IDEOptions = append(
				baseOptions.IDEOptions,
				optionName+"="+optionValue.Value,
			)
		}

		// run devpod up elsewhere
		err := client.Up(ctx, client2.UpOptions{
			CLIOptions: baseOptions,
			Debug:      cmd.Debug,

			Stdin:  stdinReader,
			Stdout: stdoutWriter,
		})
		if err != nil {
			errChan <- fmt.Errorf("executing up proxy command: %w", err)
		} else {
			errChan <- nil
		}
	}()

	// create container etc.
	result, err := tunnelserver.RunUpServer(
		cancelCtx,
		stdoutReader,
		stdinWriter,
		true,
		true,
		client.WorkspaceConfig(),
		log,
	)
	if err != nil {
		return nil, errors.Wrap(err, "run tunnel machine")
	}

	// wait until command finished
	return result, <-errChan
}

func (cmd *UpCmd) devPodUpMachine(
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.WorkspaceClient,
	log log.Logger,
) (*config2.Result, error) {
	err := startWait(ctx, client, true, log)
	if err != nil {
		return nil, err
	}

	// compress info
	workspaceInfo, wInfo, err := client.AgentInfo(cmd.CLIOptions)
	if err != nil {
		return nil, err
	}

	// create container etc.
	log.Infof("Creating devcontainer...")
	defer log.Debugf("Done creating devcontainer")

	// ssh tunnel command
	sshTunnelCmd := fmt.Sprintf("'%s' helper ssh-server --stdio", client.AgentPath())
	if log.GetLevel() == logrus.DebugLevel {
		sshTunnelCmd += " --debug"
	}

	// create agent command
	agentCommand := fmt.Sprintf(
		"'%s' agent workspace up --workspace-info '%s'",
		client.AgentPath(),
		workspaceInfo,
	)

	if log.GetLevel() == logrus.DebugLevel {
		agentCommand += " --debug"
	}

	agentInjectFunc := func(cancelCtx context.Context, sshCmd string, sshTunnelStdinReader, sshTunnelStdoutWriter *os.File, writer io.WriteCloser) error {
		return agent.InjectAgentAndExecute(
			cancelCtx,
			func(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
				return client.Command(ctx, client2.CommandOptions{
					Command: command,
					Stdin:   stdin,
					Stdout:  stdout,
					Stderr:  stderr,
				})
			},
			client.AgentLocal(),
			client.AgentPath(),
			client.AgentURL(),
			true,
			sshCmd,
			sshTunnelStdinReader,
			sshTunnelStdoutWriter,
			writer,
			log.ErrorStreamOnly(),
			wInfo.InjectTimeout,
		)
	}

	return sshtunnel.ExecuteCommand(
		ctx,
		client,
		devPodConfig.ContextOption(config.ContextOptionSSHAddPrivateKeys) == "true",
		agentInjectFunc,
		sshTunnelCmd,
		agentCommand,
		log,
		func(ctx context.Context, stdin io.WriteCloser, stdout io.Reader) (*config2.Result, error) {
			if cmd.Proxy {
				// create tunnel client on stdin & stdout
				tunnelClient, err := tunnelserver.NewTunnelClient(os.Stdin, os.Stdout, true, 0)
				if err != nil {
					return nil, errors.Wrap(err, "create tunnel client")
				}
				allowGitCredentials := devPodConfig.ContextOption(config.ContextOptionSSHInjectGitCredentials) == "true"
				allowDockerCredentials := devPodConfig.ContextOption(config.ContextOptionSSHInjectDockerCredentials) == "true"

				return tunnelserver.RunProxyServer(ctx, tunnelClient, stdout, stdin, allowGitCredentials, allowDockerCredentials, cmd.GitUsername, cmd.GitToken, log)
			}

			return tunnelserver.RunUpServer(
				ctx,
				stdout,
				stdin,
				client.AgentInjectGitCredentials(),
				client.AgentInjectDockerCredentials(),
				client.WorkspaceConfig(),
				log,
				tunnelserver.WithGitCredentialsOverride(cmd.GitUsername, cmd.GitToken),
			)
		},
	)
}

func startMarimoInBrowser(
	forwardGpg bool,
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.BaseWorkspaceClient,
	user string,
	ideOptions map[string]config.OptionValue,
	gitUsername, gitToken string,
	logger log.Logger,
) error {
	if forwardGpg {
		err := performGpgForwarding(client, logger)
		if err != nil {
			return err
		}
	}

	// determine port
	address, port, err := parseAddressAndPort(
		marimo.Options.GetValue(ideOptions, marimo.BindAddressOption),
		marimo.DefaultServerPort,
	)
	if err != nil {
		return err
	}

	// wait until reachable then open browser
	targetURL := fmt.Sprintf("http://localhost:%d?access_token=%s", port, marimo.Options.GetValue(ideOptions, marimo.AccessToken))
	if marimo.Options.GetValue(ideOptions, marimo.OpenOption) == "true" {
		go func() {
			err = open2.Open(ctx, targetURL, logger)
			if err != nil {
				logger.Errorf("error opening marimo: %v", err)
			}

			logger.Infof(
				"Successfully started marimo in browser mode. Please keep this terminal open as long as you use Marimo",
			)
		}()
	}

	// start in browser
	logger.Infof("Starting marimo in browser mode at %s", targetURL)
	extraPorts := []string{fmt.Sprintf("%s:%d", address, marimo.DefaultServerPort)}
	return startBrowserTunnel(
		ctx,
		devPodConfig,
		client,
		user,
		targetURL,
		false,
		extraPorts,
		gitUsername,
		gitToken,
		logger,
	)
}

func startJupyterNotebookInBrowser(
	forwardGpg bool,
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.BaseWorkspaceClient,
	user string,
	ideOptions map[string]config.OptionValue,
	gitUsername, gitToken string,
	logger log.Logger,
) error {
	if forwardGpg {
		err := performGpgForwarding(client, logger)
		if err != nil {
			return err
		}
	}

	// determine port
	jupyterAddress, jupyterPort, err := parseAddressAndPort(
		jupyter.Options.GetValue(ideOptions, jupyter.BindAddressOption),
		jupyter.DefaultServerPort,
	)
	if err != nil {
		return err
	}

	// wait until reachable then open browser
	targetURL := fmt.Sprintf("http://localhost:%d/lab", jupyterPort)
	if jupyter.Options.GetValue(ideOptions, jupyter.OpenOption) == "true" {
		go func() {
			err = open2.Open(ctx, targetURL, logger)
			if err != nil {
				logger.Errorf("error opening jupyter notebook: %v", err)
			}

			logger.Infof(
				"Successfully started jupyter notebook in browser mode. Please keep this terminal open as long as you use Jupyter Notebook",
			)
		}()
	}

	// start in browser
	logger.Infof("Starting jupyter notebook in browser mode at %s", targetURL)
	extraPorts := []string{fmt.Sprintf("%s:%d", jupyterAddress, jupyter.DefaultServerPort)}
	return startBrowserTunnel(
		ctx,
		devPodConfig,
		client,
		user,
		targetURL,
		false,
		extraPorts,
		gitUsername,
		gitToken,
		logger,
	)
}

func startJupyterDesktop(
	forwardGpg bool,
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.BaseWorkspaceClient,
	user string,
	ideOptions map[string]config.OptionValue,
	gitUsername, gitToken string,
	logger log.Logger,
) error {
	if forwardGpg {
		err := performGpgForwarding(client, logger)
		if err != nil {
			return err
		}
	}

	// determine port
	jupyterAddress, jupyterPort, err := parseAddressAndPort(
		jupyter.Options.GetValue(ideOptions, jupyter.BindAddressOption),
		jupyter.DefaultServerPort,
	)
	if err != nil {
		return err
	}

	// wait until reachable then open browser
	targetURL := fmt.Sprintf("http://localhost:%d/lab", jupyterPort)
	if jupyter.Options.GetValue(ideOptions, jupyter.OpenOption) == "true" {
		go func() {
			err = open2.JLabDesktop(ctx, targetURL, logger)
			if err != nil {
				logger.Errorf("error opening jupyter desktop: %v", err)
			}
			logger.Infof("Successfully started jupyter desktop")
		}()
	}

	// start in browser
	logger.Infof("Starting jupyter desktop using server %s", targetURL)
	extraPorts := []string{fmt.Sprintf("%s:%d", jupyterAddress, jupyter.DefaultServerPort)}
	return startBrowserTunnel(
		ctx,
		devPodConfig,
		client,
		user,
		targetURL,
		false,
		extraPorts,
		gitUsername,
		gitToken,
		logger,
	)
}

func startFleet(ctx context.Context, client client2.BaseWorkspaceClient, logger log.Logger) error {
	// create ssh command
	stdout := &bytes.Buffer{}
	cmd, err := createSSHCommand(
		ctx,
		client,
		logger,
		[]string{"--command", "cat " + fleet.FleetURLFile},
	)
	if err != nil {
		return err
	}
	cmd.Stdout = stdout
	err = cmd.Run()
	if err != nil {
		return command.WrapCommandError(stdout.Bytes(), err)
	}

	url := strings.TrimSpace(stdout.String())
	if len(url) == 0 {
		return fmt.Errorf("seems like fleet is not running within the container")
	}

	logger.Warnf(
		"Fleet is exposed at a publicly reachable URL, please make sure to not disclose this URL to anyone as they will be able to reach your workspace from that",
	)
	logger.Infof("Starting Fleet at %s ...", url)
	err = open.Run(url)
	if err != nil {
		return err
	}

	return nil
}

func startVSCodeInBrowser(
	forwardGpg bool,
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.BaseWorkspaceClient,
	workspaceFolder, user string,
	ideOptions map[string]config.OptionValue,
	gitUsername, gitToken string,
	logger log.Logger,
) error {
	if forwardGpg {
		err := performGpgForwarding(client, logger)
		if err != nil {
			return err
		}
	}

	// determine port
	vscodeAddress, vscodePort, err := parseAddressAndPort(
		openvscode.Options.GetValue(ideOptions, openvscode.BindAddressOption),
		openvscode.DefaultVSCodePort,
	)
	if err != nil {
		return err
	}

	// wait until reachable then open browser
	targetURL := fmt.Sprintf("http://localhost:%d/?folder=%s", vscodePort, workspaceFolder)
	if openvscode.Options.GetValue(ideOptions, openvscode.OpenOption) == "true" {
		go func() {
			err = open2.Open(ctx, targetURL, logger)
			if err != nil {
				logger.Errorf("error opening vscode: %v", err)
			}

			logger.Infof(
				"Successfully started vscode in browser mode. Please keep this terminal open as long as you use VSCode browser version",
			)
		}()
	}

	// start in browser
	logger.Infof("Starting vscode in browser mode at %s", targetURL)
	forwardPorts := openvscode.Options.GetValue(ideOptions, openvscode.ForwardPortsOption) == "true"
	extraPorts := []string{fmt.Sprintf("%s:%d", vscodeAddress, openvscode.DefaultVSCodePort)}
	return startBrowserTunnel(
		ctx,
		devPodConfig,
		client,
		user,
		targetURL,
		forwardPorts,
		extraPorts,
		gitUsername,
		gitToken,
		logger,
	)
}

func parseAddressAndPort(bindAddressOption string, defaultPort int) (string, int, error) {
	var (
		err      error
		address  string
		portName int
	)
	if bindAddressOption == "" {
		portName, err = port.FindAvailablePort(defaultPort)
		if err != nil {
			return "", 0, err
		}

		address = fmt.Sprintf("%d", portName)
	} else {
		address = bindAddressOption
		_, port, err := net.SplitHostPort(address)
		if err != nil {
			return "", 0, fmt.Errorf("parse host:port: %w", err)
		} else if port == "" {
			return "", 0, fmt.Errorf("parse ADDRESS: expected host:port, got %s", address)
		}

		portName, err = strconv.Atoi(port)
		if err != nil {
			return "", 0, fmt.Errorf("parse host:port: %w", err)
		}
	}

	return address, portName, nil
}

func startBrowserTunnel(
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.BaseWorkspaceClient,
	user, targetURL string,
	forwardPorts bool,
	extraPorts []string,
	gitUsername, gitToken string,
	logger log.Logger,
) error {
	err := tunnel.NewTunnel(
		ctx,
		func(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
			writer := logger.Writer(logrus.DebugLevel, false)
			defer writer.Close()

			cmd, err := createSSHCommand(ctx, client, logger, []string{
				"--log-output=raw",
				"--stdio",
			})
			if err != nil {
				return err
			}
			cmd.Stdout = stdout
			cmd.Stdin = stdin
			cmd.Stderr = writer
			return cmd.Run()
		},
		func(ctx context.Context, containerClient *ssh.Client) error {
			// print port to console
			streamLogger, ok := logger.(*log.StreamLogger)
			if ok {
				streamLogger.JSON(logrus.InfoLevel, map[string]string{
					"url":  targetURL,
					"done": "true",
				})
			}

			// run in container
			err := tunnel.RunInContainer(
				ctx,
				devPodConfig,
				containerClient,
				user,
				forwardPorts,
				extraPorts,
				gitUsername,
				gitToken,
				logger,
			)
			if err != nil {
				logger.Errorf("error running credentials server: %v", err)
			}

			<-ctx.Done()
			return nil
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func configureSSH(c *config.Config, client client2.BaseWorkspaceClient, sshConfigPath, user, workdir string, gpgagent bool, devPodHome string) error {
	path, err := devssh.ResolveSSHConfigPath(sshConfigPath)
	if err != nil {
		return errors.Wrap(err, "Invalid ssh config path")
	}
	sshConfigPath = path

	err = devssh.ConfigureSSHConfig(
		sshConfigPath,
		client.Context(),
		client.Workspace(),
		user,
		workdir,
		gpgagent,
		devPodHome,
		log.Default,
	)
	if err != nil {
		return err
	}

	return nil
}

func mergeDevPodUpOptions(baseOptions *provider2.CLIOptions) error {
	oldOptions := *baseOptions
	found, err := clientimplementation.DecodeOptionsFromEnv(
		clientimplementation.DevPodFlagsUp,
		baseOptions,
	)
	if err != nil {
		return fmt.Errorf("decode up options: %w", err)
	} else if found {
		baseOptions.WorkspaceEnv = append(oldOptions.WorkspaceEnv, baseOptions.WorkspaceEnv...)
		baseOptions.InitEnv = append(oldOptions.InitEnv, baseOptions.InitEnv...)
		baseOptions.PrebuildRepositories = append(oldOptions.PrebuildRepositories, baseOptions.PrebuildRepositories...)
		baseOptions.IDEOptions = append(oldOptions.IDEOptions, baseOptions.IDEOptions...)
	}

	return nil
}

func mergeEnvFromFiles(baseOptions *provider2.CLIOptions) error {
	var variables []string
	for _, file := range baseOptions.WorkspaceEnvFile {
		envFromFile, err := config2.ParseKeyValueFile(file)
		if err != nil {
			return err
		}
		variables = append(variables, envFromFile...)
	}
	baseOptions.WorkspaceEnv = append(baseOptions.WorkspaceEnv, variables...)

	return nil
}

func createSSHCommand(
	ctx context.Context,
	client client2.BaseWorkspaceClient,
	logger log.Logger,
	extraArgs []string,
) (*exec.Cmd, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	args := []string{
		"ssh",
		"--user=root",
		"--agent-forwarding=false",
		"--start-services=false",
		"--context",
		client.Context(),
		client.Workspace(),
	}
	if logger.GetLevel() == logrus.DebugLevel {
		args = append(args, "--debug")
	}
	args = append(args, extraArgs...)

	return exec.CommandContext(ctx, execPath, args...), nil
}

func setupDotfiles(
	dotfiles, script string,
	client client2.BaseWorkspaceClient,
	devPodConfig *config.Config,
	log log.Logger,
) error {
	dotfilesRepo := devPodConfig.ContextOption(config.ContextOptionDotfilesURL)
	if dotfiles != "" {
		dotfilesRepo = dotfiles
	}

	dotfilesScript := devPodConfig.ContextOption(config.ContextOptionDotfilesScript)
	if script != "" {
		dotfilesScript = script
	}

	if dotfilesRepo == "" {
		log.Debug("No dotfiles repo specified, skipping")
		return nil
	}

	log.Infof("Dotfiles git repository %s specified", dotfilesRepo)
	log.Debug("Cloning dotfiles into the devcontainer...")

	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	agentArguments := []string{
		"agent",
		"workspace",
		"install-dotfiles",
		"--repository",
		dotfilesRepo,
	}

	if log.GetLevel() == logrus.DebugLevel {
		agentArguments = append(agentArguments, "--debug")
	}

	if dotfilesScript != "" {
		log.Infof("Dotfiles script %s specified", dotfilesScript)

		agentArguments = append(agentArguments, "--install-script")
		agentArguments = append(agentArguments, dotfilesScript)
	}

	remoteUser, err := devssh.GetUser(client.WorkspaceConfig().ID, client.WorkspaceConfig().SSHConfigPath)
	if err != nil {
		remoteUser = "root"
	}

	dotCmd := exec.Command(
		execPath,
		"ssh",
		"--agent-forwarding=true",
		"--start-services=true",
		"--user",
		remoteUser,
		"--context",
		client.Context(),
		client.Workspace(),
		"--log-output=raw",
		"--command",
		agent.ContainerDevPodHelperLocation+" "+strings.Join(agentArguments, " "),
	)

	if log.GetLevel() == logrus.DebugLevel {
		dotCmd.Args = append(dotCmd.Args, "--debug")
	}

	log.Debugf("Running command: %v", dotCmd.Args)

	writer := log.Writer(logrus.InfoLevel, false)

	dotCmd.Stdout = writer
	dotCmd.Stderr = writer

	err = dotCmd.Run()
	if err != nil {
		return err
	}

	log.Infof("Done setting up dotfiles into the devcontainer")

	return nil
}

func setupGitSSHSignature(signingKey string, client client2.BaseWorkspaceClient, log log.Logger) error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	remoteUser, err := devssh.GetUser(client.WorkspaceConfig().ID, client.WorkspaceConfig().SSHConfigPath)
	if err != nil {
		remoteUser = "root"
	}

	err = exec.Command(
		execPath,
		"ssh",
		"--agent-forwarding=true",
		"--start-services=true",
		"--user",
		remoteUser,
		"--context",
		client.Context(),
		client.Workspace(),
		"--command", fmt.Sprintf("devpod agent git-ssh-signature-helper %s", signingKey),
	).Run()
	if err != nil {
		log.Error("failure in setting up git ssh signature helper")
	}
	return nil
}

func setupLoftPlatformAccess(context, provider, user string, client client2.BaseWorkspaceClient, log log.Logger) error {
	log.Infof("Setting up platform access")
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	port, err := credentials.GetPort()
	if err != nil {
		return fmt.Errorf("get port: %w", err)
	}

	command := fmt.Sprintf("\"%s\" agent container setup-loft-platform-access --context %s --provider %s --port %d", agent.ContainerDevPodHelperLocation, context, provider, port)

	log.Debugf("Executing command: %v", command)
	var errb bytes.Buffer
	cmd := exec.Command(
		execPath,
		"ssh",
		"--start-services=true",
		"--user",
		user,
		"--context",
		client.Context(),
		client.Workspace(),
		"--command", command,
	)
	cmd.Stderr = &errb
	err = cmd.Run()
	if err != nil {
		log.Debugf("failed to set up platform access in workspace: %s", errb.String())
	}

	return nil
}

func performGpgForwarding(
	client client2.BaseWorkspaceClient,
	log log.Logger,
) error {
	log.Debug("gpg forwarding enabled, performing immediately")

	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	remoteUser, err := devssh.GetUser(client.WorkspaceConfig().ID, client.WorkspaceConfig().SSHConfigPath)
	if err != nil {
		remoteUser = "root"
	}

	log.Info("forwarding gpg-agent")

	// perform in background an ssh command forwarding the
	// gpg agent, in order to have it immediately take effect
	go func() {
		err = exec.Command(
			execPath,
			"ssh",
			"--gpg-agent-forwarding=true",
			"--agent-forwarding=true",
			"--start-services=true",
			"--user",
			remoteUser,
			"--context",
			client.Context(),
			client.Workspace(),
			"--log-output=raw",
			"--command", "sleep infinity",
		).Run()
		if err != nil {
			log.Error("failure in forwarding gpg-agent")
		}
	}()

	return nil
}

// checkProviderUpdate currently only ensures the local provider is in sync with the remote for DevPod Pro instances
// Potentially auto-upgrade other providers in the future.
func checkProviderUpdate(devPodConfig *config.Config, proInstance *provider2.ProInstance, log log.Logger) error {
	if version.GetVersion() == version.DevVersion {
		log.Debugf("Skipping provider upgrade check during development")
		return nil
	}
	if proInstance == nil {
		log.Debugf("No pro instance available, skipping provider upgrade check")
		return nil
	}

	// compare versions
	newVersion, err := loft.GetProInstanceDevPodVersion(proInstance)
	if err != nil {
		return fmt.Errorf("version for pro instance %s: %w", proInstance.Host, err)
	}

	p, err := workspace2.FindProvider(devPodConfig, proInstance.Provider, log)
	if err != nil {
		return fmt.Errorf("get provider config for pro provider %s: %w", proInstance.Provider, err)
	}
	if p.Config.Version == version.DevVersion {
		return nil
	}

	v1, err := semver.Parse(strings.TrimPrefix(newVersion, "v"))
	if err != nil {
		return fmt.Errorf("parse version %s: %w", newVersion, err)
	}
	v2, err := semver.Parse(strings.TrimPrefix(p.Config.Version, "v"))
	if err != nil {
		return fmt.Errorf("parse version %s: %w", p.Config.Version, err)
	}
	if v1.Compare(v2) == 0 {
		return nil
	}
	log.Infof("New provider version available, attempting to update %s", proInstance.Provider)

	providerSource, err := workspace2.ResolveProviderSource(devPodConfig, proInstance.Provider, log)
	if err != nil {
		return fmt.Errorf("resolve provider source %s: %w", proInstance.Provider, err)
	}

	splitted := strings.Split(providerSource, "@")
	if len(splitted) == 0 {
		return fmt.Errorf("no provider source found %s", providerSource)
	}
	providerSource = splitted[0] + "@" + newVersion

	_, err = workspace2.UpdateProvider(devPodConfig, proInstance.Provider, providerSource, log)
	if err != nil {
		return fmt.Errorf("update provider %s: %w", proInstance.Provider, err)
	}

	log.Donef("Successfully updated provider %s", proInstance.Provider)
	return nil
}

func getProInstance(devPodConfig *config.Config, providerName string, log log.Logger) *provider2.ProInstance {
	proInstances, err := workspace2.ListProInstances(devPodConfig, log)
	if err != nil {
		return nil
	} else if len(proInstances) == 0 {
		return nil
	}

	proInstance, ok := workspace2.FindProviderProInstance(proInstances, providerName)
	if !ok {
		return nil
	}

	return proInstance
}
