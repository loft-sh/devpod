package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/agent/tunnelserver"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	config2 "github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/sshtunnel"
	"github.com/loft-sh/devpod/pkg/dotfiles"
	"github.com/loft-sh/devpod/pkg/ide/browseride"
	"github.com/loft-sh/devpod/pkg/ide/fleet"
	"github.com/loft-sh/devpod/pkg/ide/jetbrains"
	"github.com/loft-sh/devpod/pkg/ide/vscode"
	"github.com/loft-sh/devpod/pkg/pro"
	"github.com/loft-sh/devpod/pkg/provider"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// UpCmd holds the up cmd flags
type UpCmd struct {
	provider2.CLIOptions
	*flags.GlobalFlags

	Machine string

	ProviderOptions []string

	ConfigureSSH       bool
	GPGAgentForwarding bool
	OpenIDE            bool

	SSHConfigPath string

	DotfilesSource string
	DotfilesScript string

	Log log.Logger
}

// NewUpCmd creates a new up command
func NewUpCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &UpCmd{
		GlobalFlags: flags,
		Log:         log.Default,
	}
	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Starts a new workspace",
		RunE: func(c *cobra.Command, args []string) error {
			return cmd.Run(c.Context(), args)
		},
	}

	upCmd.Flags().BoolVar(&cmd.ConfigureSSH, "configure-ssh", true, "If true will configure the ssh config to include the DevPod workspace")
	upCmd.Flags().BoolVar(&cmd.GPGAgentForwarding, "gpg-agent-forwarding", false, "If true forward the local gpg-agent to the DevPod workspace")
	upCmd.Flags().StringVar(&cmd.SSHConfigPath, "ssh-config", "", "The path to the ssh config to modify, if empty will use ~/.ssh/config")
	upCmd.Flags().StringVar(&cmd.DotfilesSource, "dotfiles", "", "The path or url to the dotfiles to use in the container")
	upCmd.Flags().StringVar(&cmd.DotfilesScript, "dotfiles-script", "", "The path in dotfiles directory to use to install the dotfiles, if empty will try to guess")
	upCmd.Flags().StringArrayVar(&cmd.IDEOptions, "ide-option", []string{}, "IDE option in the form KEY=VALUE")
	upCmd.Flags().StringVar(&cmd.DevContainerImage, "devcontainer-image", "", "The container image to use, this will override the devcontainer.json value in the project")
	upCmd.Flags().StringVar(&cmd.DevContainerPath, "devcontainer-path", "", "The path to the devcontainer.json relative to the project")
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
	upCmd.Flags().StringVar(&cmd.GitBranch, "git-branch", "", "The git branch to use")
	upCmd.Flags().StringVar(&cmd.GitCommit, "git-commit", "", "The git commit SHA to use")
	upCmd.Flags().Var(&cmd.GitCloneStrategy, "git-clone-strategy", "The git clone strategy DevPod uses to checkout git based workspaces. Can be full (default), blobless, treeless or shallow")
	upCmd.Flags().StringVar(&cmd.GitSSHSigningKey, "git-ssh-signing-key", "", "The ssh key to use when signing git commits. Used to explicitly setup DevPod's ssh signature forwarding with given key. Should be same format as value of `git config user.signingkey`")
	upCmd.Flags().StringVar(&cmd.FallbackImage, "fallback-image", "", "The fallback image to use if no devcontainer configuration has been detected")

	upCmd.Flags().BoolVar(&cmd.DisableDaemon, "disable-daemon", false, "If enabled, will not install a daemon into the target machine to track activity")
	upCmd.Flags().StringVar(&cmd.Source, "source", "", "Optional source for the workspace. E.g. git:https://github.com/my-org/my-repo")
	upCmd.Flags().BoolVar(&cmd.Pro, "pro", false, "If true will start in pro mode")
	upCmd.Flags().BoolVar(&cmd.ForceCredentials, "force-credentials", false, "If true will always use local credentials")
	_ = upCmd.Flags().MarkHidden("force-credentials")

	upCmd.Flags().StringVar(&cmd.SSHKey, "ssh-key", "", "The ssh-key to use")
	_ = upCmd.Flags().MarkHidden("ssh-key")

	// testing
	upCmd.Flags().StringVar(&cmd.DaemonInterval, "daemon-interval", "", "TESTING ONLY")
	upCmd.Flags().BoolVar(&cmd.ForceDockerless, "force-dockerless", false, "TESTING ONLY")
	_ = upCmd.Flags().MarkHidden("daemon-interval")
	_ = upCmd.Flags().MarkHidden("force-dockerless")
	return upCmd
}

// Run runs the command logic
func (cmd *UpCmd) Run(
	ctx context.Context,
	args []string,
) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	workspaceSource, err := cmd.resolveOptions(ctx, devPodConfig)
	if err != nil {
		return fmt.Errorf("resolve up options: %w", err)
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
		cmd.SSHConfigPath,
		workspaceSource,
		cmd.GitBranch,
		cmd.GitCommit,
		cmd.UID,
		true,
		cmd.Log,
	)
	if err != nil {
		return fmt.Errorf("resolve workspace: %w", err)
	}

	if cmd.Pro {
		return cmd.upServerMode(ctx, devPodConfig, client)
	}

	err = pro.UpdateProvider(devPodConfig, client.Provider(), cmd.Log)
	if err != nil {
		return err
	}

	result, err := cmd.upClientMode(ctx, devPodConfig, client, cmd.Log)
	if err != nil {
		return err
	}
	if result == nil {
		return fmt.Errorf("didn't receive a result back from agent")
	}

	remoteUser := config2.GetRemoteUser(result)

	// configure container ssh
	if cmd.ConfigureSSH {
		err = cmd.setupSSHConfig(devPodConfig, client, result, remoteUser)
		if err != nil {
			return fmt.Errorf("setup local ssh config: %w", err)
		}

		cmd.Log.Infof("Run 'ssh %s.devpod' to ssh into the devcontainer", client.Workspace())
	}

	// setup dotfiles in the container
	err = dotfiles.Setup(cmd.DotfilesSource, cmd.DotfilesScript, client, devPodConfig, cmd.Log)
	if err != nil {
		return err
	}

	// setup git ssh signature
	if cmd.GitSSHSigningKey != "" {
		err = cmd.setupGitSSHSignature(client)
		if err != nil {
			return fmt.Errorf("setup git ssh signing: %w", err)
		}
	}

	// open IDE
	if cmd.OpenIDE {
		err = cmd.connectToIDE(ctx, devPodConfig, client, result, remoteUser)
		if err != nil {
			return fmt.Errorf("connect to IDE: %w", err)
		}
	}

	return nil
}

func (cmd *UpCmd) resolveOptions(ctx context.Context, devPodConfig *config.Config) (*provider.WorkspaceSource, error) {
	// try to parse flags from env
	err := mergeDevPodUpOptions(&cmd.CLIOptions)
	if err != nil {
		return nil, fmt.Errorf("merge from environment: %w", err)
	}
	err = mergeEnvFromFiles(&cmd.CLIOptions)
	if err != nil {
		return nil, fmt.Errorf("merge from files: %w", err)
	}

	// a reset implies a recreate
	if cmd.Reset {
		cmd.Recreate = true
	}

	if cmd.Pro {
		cmd.Log = cmd.Log.ErrorStreamOnly()
		cmd.Log.Debugf("Using error stream as --proxy is enabled")
	}

	var source *provider2.WorkspaceSource
	if cmd.Source != "" {
		source = provider2.ParseWorkspaceSource(cmd.Source)
		if source == nil {
			return nil, fmt.Errorf("workspace source is missing")
		}
	}

	if cmd.SSHConfigPath == "" {
		cmd.SSHConfigPath = devPodConfig.ContextOption(config.ContextOptionSSHConfigPath)
	}

	return source, nil
}

func (cmd *UpCmd) upClientMode(
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

	// check what client we have
	if workspaceClient, ok := client.(client2.WorkspaceClient); ok {
		result, err = cmd.upMachine(ctx, devPodConfig, workspaceClient, log)
		if err != nil {
			return nil, err
		}
	} else if proxyClient, ok := client.(client2.ProxyClient); ok {
		result, err = cmd.upProxy(ctx, proxyClient, log)
		if err != nil {
			return nil, err
		}
	}

	// save result to file
	err = provider2.SaveWorkspaceResult(client.WorkspaceConfig(), result)
	if err != nil {
		return nil, fmt.Errorf("save workspace result: %w", err)
	}

	return result, nil
}

func (cmd *UpCmd) upServerMode(ctx context.Context, devPodConfig *config.Config, client client2.BaseWorkspaceClient) error {
	workspaceClient, ok := client.(client2.WorkspaceClient)
	if !ok {
		return fmt.Errorf("expected workspace client, got %T", client)
	}

	err := client.Lock(ctx)
	if err != nil {
		return err
	}
	defer client.Unlock()

	_, err = cmd.upMachine(ctx, devPodConfig, workspaceClient, cmd.Log)
	return err
}

func (cmd *UpCmd) upProxy(
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

func (cmd *UpCmd) upMachine(
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
	workspaceInfo, _, err := client.AgentInfo(cmd.CLIOptions)
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
			if cmd.Pro {
				// create tunnel client on stdin & stdout
				tunnelClient, err := tunnelserver.NewTunnelClient(os.Stdin, os.Stdout, true, 0)
				if err != nil {
					return nil, errors.Wrap(err, "create tunnel client")
				}

				return tunnelserver.RunProxyServer(ctx, tunnelClient, stdout, stdin, log, cmd.GitUsername, cmd.GitToken)
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

func (cmd *UpCmd) setupSSHConfig(devPodConfig *config.Config, client client2.BaseWorkspaceClient, result *config2.Result, user string) error {
	var workdir string
	if result.MergedConfig != nil && result.MergedConfig.WorkspaceFolder != "" {
		workdir = result.MergedConfig.WorkspaceFolder
	}

	if client.WorkspaceConfig().Source.GitSubPath != "" {
		result.SubstitutionContext.ContainerWorkspaceFolder = filepath.Join(result.SubstitutionContext.ContainerWorkspaceFolder, client.WorkspaceConfig().Source.GitSubPath)
		workdir = result.SubstitutionContext.ContainerWorkspaceFolder
	}

	forwardGPGAgent := cmd.GPGAgentForwarding || devPodConfig.ContextOption(config.ContextOptionGPGAgentForwarding) == "true"

	sshConfigPath, err := devssh.ResolveSSHConfigPath(cmd.SSHConfigPath)
	if err != nil {
		return errors.Wrap(err, "Invalid ssh config path")
	}

	return devssh.ConfigureSSHConfig(
		sshConfigPath,
		client.Context(),
		client.Workspace(),
		user,
		workdir,
		forwardGPGAgent,
		log.Default,
	)
}

func (cmd *UpCmd) setupGitSSHSignature(client client2.BaseWorkspaceClient) error {
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
		"--command", fmt.Sprintf("devpod agent git-ssh-signature-helper %s", cmd.GitSSHSigningKey),
	).Run()
	if err != nil {
		cmd.Log.Error("failure in setting up git ssh signature helper")
	}
	return nil
}

func (cmd *UpCmd) connectToIDE(ctx context.Context, devPodConfig *config.Config, client client2.BaseWorkspaceClient, result *config2.Result, user string) error {
	ideConfig := client.WorkspaceConfig().IDE
	switch ideConfig.Name {
	case string(config.IDEVSCode):
		return vscode.Open(
			ctx,
			client.Workspace(),
			result.SubstitutionContext.ContainerWorkspaceFolder,
			vscode.Options.GetValue(ideConfig.Options, vscode.OpenNewWindow) == "true",
			vscode.ReleaseChannelStable,
			cmd.Log,
		)
	case string(config.IDEVSCodeInsiders):
		return vscode.Open(
			ctx,
			client.Workspace(),
			result.SubstitutionContext.ContainerWorkspaceFolder,
			vscode.Options.GetValue(ideConfig.Options, vscode.OpenNewWindow) == "true",
			vscode.ReleaseChannelInsiders,
			cmd.Log,
		)
	case string(config.IDERustRover):
		return jetbrains.NewRustRoverServer(config2.GetRemoteUser(result), ideConfig.Options, cmd.Log).
			OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case string(config.IDEGoland):
		return jetbrains.NewGolandServer(config2.GetRemoteUser(result), ideConfig.Options, cmd.Log).
			OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case string(config.IDEPyCharm):
		return jetbrains.NewPyCharmServer(config2.GetRemoteUser(result), ideConfig.Options, cmd.Log).
			OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case string(config.IDEPhpStorm):
		return jetbrains.NewPhpStorm(config2.GetRemoteUser(result), ideConfig.Options, cmd.Log).
			OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case string(config.IDEIntellij):
		return jetbrains.NewIntellij(config2.GetRemoteUser(result), ideConfig.Options, cmd.Log).
			OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case string(config.IDECLion):
		return jetbrains.NewCLionServer(config2.GetRemoteUser(result), ideConfig.Options, cmd.Log).
			OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case string(config.IDERider):
		return jetbrains.NewRiderServer(config2.GetRemoteUser(result), ideConfig.Options, cmd.Log).
			OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case string(config.IDERubyMine):
		return jetbrains.NewRubyMineServer(config2.GetRemoteUser(result), ideConfig.Options, cmd.Log).
			OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case string(config.IDEWebStorm):
		return jetbrains.NewWebStormServer(config2.GetRemoteUser(result), ideConfig.Options, cmd.Log).
			OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case string(config.IDEFleet):
		return fleet.Start(ctx, client, cmd.Log)
	case string(config.IDEOpenVSCode):
		return browseride.StartVSCode(
			cmd.GPGAgentForwarding,
			ctx,
			devPodConfig,
			client,
			result.SubstitutionContext.ContainerWorkspaceFolder,
			user,
			ideConfig.Options,
			cmd.GitUsername,
			cmd.GitToken,
			cmd.Log,
		)
	case string(config.IDEJupyterNotebook):
		return browseride.StartJupyterNotebook(
			cmd.GPGAgentForwarding,
			ctx,
			devPodConfig,
			client,
			user,
			ideConfig.Options,
			cmd.GitUsername,
			cmd.GitToken,
			cmd.Log,
		)
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
