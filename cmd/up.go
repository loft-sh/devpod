package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/agent/tunnelserver"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	config2 "github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/ide/fleet"
	"github.com/loft-sh/devpod/pkg/ide/jetbrains"
	"github.com/loft-sh/devpod/pkg/ide/jupyter"
	"github.com/loft-sh/devpod/pkg/ide/openvscode"
	"github.com/loft-sh/devpod/pkg/ide/vscode"
	open2 "github.com/loft-sh/devpod/pkg/open"
	"github.com/loft-sh/devpod/pkg/port"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/tunnel"
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

	ConfigureSSH bool
	OpenIDE      bool

	SSHConfigPath string

	DotfilesSource string
	DotfilesScript string
}

// NewUpCmd creates a new up command
func NewUpCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &UpCmd{
		GlobalFlags: flags,
	}
	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Starts a new workspace",
		RunE: func(_ *cobra.Command, args []string) error {
			// try to parse flags from env
			err := mergeDevPodUpOptions(&cmd.CLIOptions)
			if err != nil {
				return err
			}

			ctx := context.Background()
			var logger log.Logger = log.Default
			if cmd.Proxy {
				logger = logger.ErrorStreamOnly()
				logger.Debugf("Using error stream as --proxy is enabled")
			}

			devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
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
				source,
				true,
				logger,
			)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, devPodConfig, client, logger)
		},
	}

	upCmd.Flags().BoolVar(&cmd.ConfigureSSH, "configure-ssh", true, "If true will configure the ssh config to include the DevPod workspace")
	upCmd.Flags().StringVar(&cmd.SSHConfigPath, "ssh-config", "", "The path to the ssh config to modify, if empty will use ~/.ssh/config")
	upCmd.Flags().StringVar(&cmd.DotfilesSource, "dotfiles", "", "The path or url to the dotfiles to use in the container")
	upCmd.Flags().StringVar(&cmd.DotfilesScript, "dotfiles-script", "", "The path in dotfiles directory to use to install the dotfiles, if empty will try to guess")
	upCmd.Flags().StringArrayVar(&cmd.IDEOptions, "ide-option", []string{}, "IDE option in the form KEY=VALUE")
	upCmd.Flags().StringVar(&cmd.DevContainerImage, "devcontainer-image", "", "The container image to use, this will override the devcontainer.json value in the project")
	upCmd.Flags().StringVar(&cmd.DevContainerPath, "devcontainer-path", "", "The path to the devcontainer.json relative to the project")
	upCmd.Flags().StringArrayVar(&cmd.ProviderOptions, "provider-option", []string{}, "Provider option in the form KEY=VALUE")
	upCmd.Flags().BoolVar(&cmd.Recreate, "recreate", false, "If true will remove any existing containers and recreate them")
	upCmd.Flags().StringSliceVar(&cmd.PrebuildRepositories, "prebuild-repository", []string{}, "Docker repository that hosts devpod prebuilds for this workspace")
	upCmd.Flags().StringArrayVar(&cmd.WorkspaceEnv, "workspace-env", []string{}, "Extra env variables to put into the workspace. E.g. MY_ENV_VAR=MY_VALUE")
	upCmd.Flags().StringVar(&cmd.ID, "id", "", "The id to use for the workspace")
	upCmd.Flags().StringVar(&cmd.Machine, "machine", "", "The machine to use for this workspace. The machine needs to exist beforehand or the command will fail. If the workspace already exists, this option has no effect")
	upCmd.Flags().StringVar(&cmd.IDE, "ide", "", "The IDE to open the workspace in. If empty will use vscode locally or in browser")
	upCmd.Flags().BoolVar(&cmd.OpenIDE, "open-ide", true, "If this is false and an IDE is configured, DevPod will only install the IDE server backend, but not open it")

	upCmd.Flags().BoolVar(&cmd.DisableDaemon, "disable-daemon", false, "If enabled, will not install a daemon into the target machine to track activity")
	upCmd.Flags().StringVar(&cmd.Source, "source", "", "Optional source for the workspace. E.g. git:https://github.com/my-org/my-repo")
	upCmd.Flags().BoolVar(&cmd.Proxy, "proxy", false, "If true will forward agent requests to stdio")

	// testing
	upCmd.Flags().StringVar(&cmd.DaemonInterval, "daemon-interval", "", "TESTING ONLY")
	_ = upCmd.Flags().MarkHidden("daemon-interval")
	return upCmd
}

// Run runs the command logic
func (cmd *UpCmd) Run(
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.BaseWorkspaceClient,
	log log.Logger,
) error {
	// run devpod agent up
	result, err := cmd.devPodUp(ctx, client, log)
	if err != nil {
		return err
	} else if result == nil {
		return fmt.Errorf("didn't receive a result back from agent")
	} else if cmd.Proxy {
		return nil
	}

	// get user from result
	user := config2.GetRemoteUser(result)

	// configure container ssh
	if cmd.ConfigureSSH {
		err = configureSSH(client, cmd.SSHConfigPath, user)
		if err != nil {
			return err
		}

		log.Infof("Run 'ssh %s.devpod' to ssh into the devcontainer", client.Workspace())
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
				log,
			)
		case string(config.IDEOpenVSCode):
			return startVSCodeInBrowser(
				ctx,
				devPodConfig,
				client,
				result.SubstitutionContext.ContainerWorkspaceFolder,
				user,
				ideConfig.Options,
				log,
			)
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
				ctx,
				devPodConfig,
				client,
				user,
				ideConfig.Options,
				log,
			)
		}
	}

	return nil
}

func (cmd *UpCmd) devPodUp(
	ctx context.Context,
	client client2.BaseWorkspaceClient,
	log log.Logger,
) (*config2.Result, error) {
	err := client.Lock(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Unlock()

	// check if regular workspace client
	workspaceClient, ok := client.(client2.WorkspaceClient)
	if ok {
		return cmd.devPodUpMachine(ctx, workspaceClient, log)
	}

	// check if proxy client
	proxyClient, ok := client.(client2.ProxyClient)
	if ok {
		return cmd.devPodUpProxy(ctx, proxyClient, log)
	}

	return nil, nil
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
	result, err := tunnelserver.RunTunnelServer(
		cancelCtx,
		stdoutReader,
		stdinWriter,
		true,
		true,
		client.WorkspaceConfig(),
		nil,
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
	client client2.WorkspaceClient,
	log log.Logger,
) (*config2.Result, error) {
	err := startWait(ctx, client, true, log)
	if err != nil {
		return nil, err
	}

	// compress info
	workspaceInfo, _, err := client.AgentInfo(cmd.CLIOptions)
	if err != nil {
		return nil, err
	}

	// create container etc.
	log.Infof("Creating devcontainer...")
	defer log.Debugf("Done creating devcontainer")
	command := fmt.Sprintf(
		"'%s' agent workspace up --workspace-info '%s'",
		client.AgentPath(),
		workspaceInfo,
	)
	if log.GetLevel() == logrus.DebugLevel {
		command += " --debug"
	}

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

	errChan := make(chan error, 1)
	go func() {
		defer log.Debugf("Done executing up command")
		defer cancel()

		writer := log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		log.Debugf("Inject and run command: %s", command)
		err := agent.InjectAgentAndExecute(
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
			command,
			stdinReader,
			stdoutWriter,
			writer,
			log.ErrorStreamOnly(),
		)
		if err != nil {
			errChan <- fmt.Errorf("executing agent command: %w", err)
		} else {
			errChan <- nil
		}
	}()

	// create container etc.
	var result *config2.Result
	if cmd.Proxy {
		// create client on stdin & stdout
		tunnelClient, err := tunnelserver.NewTunnelClient(os.Stdin, os.Stdout, true)
		if err != nil {
			return nil, errors.Wrap(err, "create tunnel client")
		}

		// create proxy server
		result, err = tunnelserver.RunProxyServer(
			cancelCtx,
			tunnelClient,
			stdoutReader,
			stdinWriter,
			log,
		)
		if err != nil {
			return nil, errors.Wrap(err, "run proxy tunnel")
		}
	} else {
		result, err = tunnelserver.RunTunnelServer(
			cancelCtx,
			stdoutReader,
			stdinWriter,
			client.AgentInjectGitCredentials(),
			client.AgentInjectDockerCredentials(),
			client.WorkspaceConfig(),
			nil,
			log,
		)
		if err != nil {
			return nil, errors.Wrap(err, "run tunnel machine")
		}
	}

	// wait until command finished
	return result, <-errChan
}

func startJupyterNotebookInBrowser(
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.BaseWorkspaceClient,
	user string,
	ideOptions map[string]config.OptionValue,
	logger log.Logger,
) error {
	// determine port
	jupyterAddress, jupyterPort, err := parseAddressAndPort(
		jupyter.Options.GetValue(ideOptions, jupyter.BindAddressOption),
		jupyter.DefaultServerPort,
	)
	if err != nil {
		return err
	}

	// wait until reachable then open browser
	targetURL := fmt.Sprintf("http://localhost:%d", jupyterPort)
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
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.BaseWorkspaceClient,
	workspaceFolder, user string,
	ideOptions map[string]config.OptionValue,
	logger log.Logger,
) error {
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
				true,
				true,
				extraPorts,
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

func configureSSH(client client2.BaseWorkspaceClient, configPath, user string) error {
	err := devssh.ConfigureSSHConfig(
		configPath,
		client.Context(),
		client.Workspace(),
		user,
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
		baseOptions.PrebuildRepositories = append(oldOptions.PrebuildRepositories, baseOptions.PrebuildRepositories...)
		baseOptions.IDEOptions = append(oldOptions.IDEOptions, baseOptions.IDEOptions...)
	}

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

	dotCmd := exec.Command(
		execPath,
		"ssh",
		"--agent-forwarding=true",
		"--start-services=false",
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
