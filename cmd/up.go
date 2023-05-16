package cmd

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	config2 "github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/ide/jetbrains"
	"github.com/loft-sh/devpod/pkg/ide/openvscode"
	"github.com/loft-sh/devpod/pkg/log"
	open2 "github.com/loft-sh/devpod/pkg/open"
	"github.com/loft-sh/devpod/pkg/port"
	"github.com/loft-sh/devpod/pkg/tunnel"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

// UpCmd holds the up cmd flags
type UpCmd struct {
	*flags.GlobalFlags

	ID      string
	Machine string

	IDE        string
	IDEOptions []string

	ProviderOptions      []string
	PrebuildRepositories []string

	DevContainerPath string

	Recreate bool
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
			ctx := context.Background()
			devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
			if err != nil {
				return err
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
				cmd.DevContainerPath,
				true,
				log.Default,
			)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, devPodConfig, client)
		},
	}

	upCmd.Flags().StringSliceVar(&cmd.IDEOptions, "ide-option", []string{}, "IDE option in the form KEY=VALUE")
	upCmd.Flags().StringVar(&cmd.DevContainerPath, "devcontainer-path", "", "The path to the devcontainer.json relative to the project")
	upCmd.Flags().StringSliceVar(&cmd.ProviderOptions, "provider-option", []string{}, "Provider option in the form KEY=VALUE")
	upCmd.Flags().BoolVar(&cmd.Recreate, "recreate", false, "If true will remove any existing containers and recreate them")
	upCmd.Flags().StringSliceVar(&cmd.PrebuildRepositories, "prebuild-repository", []string{}, "Docker repository that hosts devpod prebuilds for this workspace")
	upCmd.Flags().StringVar(&cmd.ID, "id", "", "The id to use for the workspace")
	upCmd.Flags().StringVar(&cmd.Machine, "machine", "", "The machine to use for this workspace. The machine needs to exist beforehand or the command will fail. If the workspace already exists, this option has no effect")
	upCmd.Flags().StringVar(&cmd.IDE, "ide", "", "The IDE to open the workspace in. If empty will use vscode locally or in browser")
	return upCmd
}

// Run runs the command logic
func (cmd *UpCmd) Run(ctx context.Context, devPodConfig *config.Config, client client2.WorkspaceClient) error {
	// run devpod agent up
	result, err := cmd.devPodUp(ctx, client, log.Default)
	if err != nil {
		return err
	} else if result == nil {
		return fmt.Errorf("didn't receive a result back from agent")
	}

	// get user from result
	user := config2.GetRemoteUser(result)

	// configure container ssh
	err = configureSSH(client, user)
	if err != nil {
		return err
	}
	log.Default.Infof("Run 'ssh %s.devpod' to ssh into the devcontainer", client.Workspace())

	// open ide
	ideConfig := client.WorkspaceConfig().IDE
	switch ideConfig.Name {
	case string(config.IDEVSCode):
		return startVSCodeLocally(client, result.SubstitutionContext.ContainerWorkspaceFolder, user, log.Default)
	case string(config.IDEOpenVSCode):
		return startInBrowser(ctx, devPodConfig, client, result.SubstitutionContext.ContainerWorkspaceFolder, user, ideConfig.Options, log.Default)
	case string(config.IDEGoland):
		return jetbrains.NewGolandServer(config2.GetRemoteUser(result), ideConfig.Options, log.Default).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case string(config.IDEPyCharm):
		return jetbrains.NewPyCharmServer(config2.GetRemoteUser(result), ideConfig.Options, log.Default).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case string(config.IDEPhpStorm):
		return jetbrains.NewPhpStorm(config2.GetRemoteUser(result), ideConfig.Options, log.Default).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case string(config.IDEIntellij):
		return jetbrains.NewIntellij(config2.GetRemoteUser(result), ideConfig.Options, log.Default).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case string(config.IDECLion):
		return jetbrains.NewCLionServer(config2.GetRemoteUser(result), ideConfig.Options, log.Default).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case string(config.IDERider):
		return jetbrains.NewRiderServer(config2.GetRemoteUser(result), ideConfig.Options, log.Default).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case string(config.IDERubyMine):
		return jetbrains.NewRubyMineServer(config2.GetRemoteUser(result), ideConfig.Options, log.Default).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case string(config.IDEWebStorm):
		return jetbrains.NewWebStormServer(config2.GetRemoteUser(result), ideConfig.Options, log.Default).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	}

	return nil
}

func startVSCodeLocally(client client2.WorkspaceClient, workspaceFolder, user string, log log.Logger) error {
	log.Infof("Starting VSCode...")
	err := open.Start(`vscode://vscode-remote/ssh-remote+` + url.QueryEscape(user) + `@` + client.Workspace() + `.devpod/` + url.QueryEscape(workspaceFolder))
	if err != nil {
		return err
	}
	return nil
}

func startInBrowser(ctx context.Context, devPodConfig *config.Config, client client2.WorkspaceClient, workspaceFolder, user string, ideOptions map[string]config.OptionValue, log *log.StreamLogger) error {
	// determine port
	vscodePort, err := port.FindAvailablePort(openvscode.DefaultVSCodePort)
	if err != nil {
		return err
	}

	// wait until reachable then open browser
	targetURL := fmt.Sprintf("http://localhost:%d/?folder=%s", vscodePort, workspaceFolder)
	if openvscode.Options.GetValue(ideOptions, openvscode.OpenOption) == "true" {
		go func() {
			err = open2.Open(ctx, targetURL, log)
			if err != nil {
				log.Errorf("error opening vscode: %v", err)
			}

			log.Infof("Successfully started vscode in browser mode. Please keep this terminal open as long as you use VSCode browser version")
		}()
	}

	// print port to console
	log.JSON(logrus.InfoLevel, map[string]string{
		"url":  targetURL,
		"done": "true",
	})

	// start in browser
	log.Infof("Starting vscode in browser mode at %s", targetURL)
	err = tunnel.NewContainerTunnel(client, log).Run(ctx, nil, func(ctx context.Context, hostClient, containerClient *ssh.Client) error {
		err := tunnel.RunInContainer(ctx, client, devPodConfig, hostClient, containerClient, user, true, true, []string{fmt.Sprintf("%d:%d", vscodePort, openvscode.DefaultVSCodePort)}, log)
		if err != nil {
			log.Errorf("error running credentials server: %v", err)
		}

		<-ctx.Done()
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (cmd *UpCmd) devPodUp(ctx context.Context, client client2.WorkspaceClient, log log.Logger) (*config2.Result, error) {
	err := startWait(ctx, client, true, log)
	if err != nil {
		return nil, err
	}

	return cmd.devPodUpMachine(ctx, client, log)
}

func (cmd *UpCmd) devPodUpMachine(ctx context.Context, client client2.WorkspaceClient, log log.Logger) (*config2.Result, error) {
	// compress info
	workspaceInfo, _, err := client.AgentInfo()
	if err != nil {
		return nil, err
	}

	// create container etc.
	log.Infof("Creating devcontainer...")
	defer log.Debugf("Done creating devcontainer")
	command := fmt.Sprintf("%s agent workspace up --workspace-info '%s'", client.AgentPath(), workspaceInfo)
	if log.GetLevel() == logrus.DebugLevel {
		command += " --debug"
	}
	for _, repo := range cmd.PrebuildRepositories {
		command += fmt.Sprintf(" --prebuild-repository '%s'", repo)
	}
	if cmd.Recreate {
		command += " --recreate"
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

		writer := log.Writer(logrus.DebugLevel, false)
		defer writer.Close()

		log.Debugf("Inject and run command: %s", command)
		errChan <- agent.InjectAgentAndExecute(cancelCtx, func(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			return client.Command(ctx, client2.CommandOptions{
				Command: command,
				Stdin:   stdin,
				Stdout:  stdout,
				Stderr:  stderr,
			})
		}, client.AgentPath(), client.AgentURL(), true, command, stdinReader, stdoutWriter, writer, log.ErrorStreamOnly())
	}()

	// get workspace config
	agentConfig := client.AgentConfig()

	// create container etc.
	result, err := agent.RunTunnelServer(
		cancelCtx,
		stdoutReader,
		stdinWriter,
		false,
		string(agentConfig.InjectGitCredentials) == "true",
		string(agentConfig.InjectDockerCredentials) == "true",
		client.WorkspaceConfig(),
		log,
	)
	if err != nil {
		return nil, errors.Wrap(err, "run tunnel machine")
	}

	// wait until command finished
	return result, <-errChan
}
