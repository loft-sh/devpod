package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	config2 "github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/ide/jetbrains"
	"github.com/loft-sh/devpod/pkg/ide/openvscode"
	"github.com/loft-sh/devpod/pkg/log"
	open2 "github.com/loft-sh/devpod/pkg/open"
	"github.com/loft-sh/devpod/pkg/port"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/tunnel"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"os/exec"
)

// UpCmd holds the up cmd flags
type UpCmd struct {
	*flags.GlobalFlags

	ID      string
	Machine string
	IDE     string

	ProviderOptions []string

	PrebuildRepositories []string

	ForceBuild bool
	Recreate   bool
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
			devPodConfig, err := config.LoadConfig(cmd.Context)
			if err != nil {
				return err
			}

			ideConfig, err := cmd.parseIDE(ctx, devPodConfig, args)
			if err != nil {
				return err
			}

			client, err := workspace2.ResolveWorkspace(ctx, devPodConfig, ideConfig, args, cmd.ID, cmd.Machine, cmd.Provider, cmd.ProviderOptions, log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, client)
		},
	}

	upCmd.Flags().StringSliceVar(&cmd.ProviderOptions, "provider-option", []string{}, "Provider option in the form KEY=VALUE")
	upCmd.Flags().BoolVar(&cmd.ForceBuild, "force-build", false, "If true will rebuild the container even if there is a prebuild already")
	upCmd.Flags().BoolVar(&cmd.Recreate, "recreate", false, "If true will remove any existing containers and recreate them")
	upCmd.Flags().StringSliceVar(&cmd.PrebuildRepositories, "prebuild-repository", []string{}, "Docker respository that hosts devpod prebuilds for this workspace")
	upCmd.Flags().StringVar(&cmd.ID, "id", "", "The id to use for the workspace")
	upCmd.Flags().StringVar(&cmd.Machine, "machine", "", "The machine to use for this workspace. The machine needs to exist beforehand or the command will fail. If the workspace already exists, this option has no effect")
	upCmd.Flags().StringVar(&cmd.IDE, "ide", "", "The IDE to open the workspace in. If empty will use vscode locally or in browser")
	return upCmd
}

// Run runs the command logic
func (cmd *UpCmd) Run(ctx context.Context, client client2.WorkspaceClient) error {
	// run devpod agent up
	result, err := cmd.devPodUp(ctx, client, log.Default)
	if err != nil {
		return err
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
	switch client.WorkspaceConfig().IDE.IDE {
	case provider2.IDEVSCode:
		return startVSCodeLocally(client, result.SubstitutionContext.ContainerWorkspaceFolder, log.Default)
	case provider2.IDEOpenVSCode:
		return startInBrowser(ctx, client, user, log.Default)
	case provider2.IDEGoland:
		return jetbrains.NewGolandServer(config2.GetRemoteUser(result), log.Default).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case provider2.IDEPyCharm:
		return jetbrains.NewPyCharmServer(config2.GetRemoteUser(result), log.Default).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case provider2.IDEPhpStorm:
		return jetbrains.NewPhpStorm(config2.GetRemoteUser(result), log.Default).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case provider2.IDEIntellij:
		return jetbrains.NewIntellij(config2.GetRemoteUser(result), log.Default).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case provider2.IDECLion:
		return jetbrains.NewCLionServer(config2.GetRemoteUser(result), log.Default).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case provider2.IDERider:
		return jetbrains.NewRiderServer(config2.GetRemoteUser(result), log.Default).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case provider2.IDERubyMine:
		return jetbrains.NewRubyMineServer(config2.GetRemoteUser(result), log.Default).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	case provider2.IDEWebStorm:
		return jetbrains.NewWebStormServer(config2.GetRemoteUser(result), log.Default).OpenGateway(result.SubstitutionContext.ContainerWorkspaceFolder, client.Workspace())
	}

	return nil
}

func (cmd *UpCmd) parseIDE(ctx context.Context, devPodConfig *config.Config, args []string) (*provider2.WorkspaceIDEConfig, error) {
	if cmd.IDE == "" {
		if len(args) == 0 {
			return nil, nil
		}

		_, err := workspace2.GetWorkspace(ctx, devPodConfig, nil, args, log.Default)
		if err == nil {
			return nil, nil
		}

		cmd.IDE = string(ide.Detect())
	}

	ideStr, err := ide.Parse(cmd.IDE)
	if err != nil {
		return nil, err
	}

	return &provider2.WorkspaceIDEConfig{
		IDE: ideStr,
	}, nil
}

func startVSCodeLocally(client client2.WorkspaceClient, workspaceFolder string, log log.Logger) error {
	log.Infof("Starting VSCode...")
	err := exec.Command("code", "--folder-uri", fmt.Sprintf("vscode-remote://ssh-remote+%s.devpod/%s", client.Workspace(), workspaceFolder)).Run()
	if err != nil {
		return err
	}

	return nil
}

func startInBrowser(ctx context.Context, client client2.WorkspaceClient, user string, log log.Logger) error {
	agentClient, ok := client.(client2.AgentClient)
	if !ok {
		return fmt.Errorf("--browser is currently only supported for machine providers")
	}

	// determine port
	vscodePort, err := port.FindAvailablePort(openvscode.DefaultVSCodePort)
	if err != nil {
		return err
	}

	// wait until reachable then open browser
	go func() {
		err = open2.Open(ctx, fmt.Sprintf("http://localhost:%d/?folder=/workspaces/%s", vscodePort, client.Workspace()), log)
		if err != nil {
			log.Errorf("error opening vscode: %v", err)
		}

		log.Infof("Successfully started vscode in browser mode. Please keep this terminal open as long as you use VSCode browser version")
	}()

	// start in browser
	log.Infof("Starting vscode in browser mode...")
	err = tunnel.NewContainerTunnel(agentClient, log).Run(ctx, nil, func(client *ssh.Client) error {
		log.Debugf("Connected to container")
		go func() {
			err := runCredentialsServer(ctx, client, user, log)
			if err != nil {
				log.Errorf("error running credentials server: %v", err)
			}
		}()

		return devssh.PortForward(client, fmt.Sprintf("localhost:%d", vscodePort), fmt.Sprintf("localhost:%d", openvscode.DefaultVSCodePort), log)
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

	agentClient, ok := client.(client2.AgentClient)
	if ok {
		return cmd.devPodUpMachine(ctx, agentClient, log)
	}

	return nil, nil
}

func (cmd *UpCmd) devPodUpMachine(ctx context.Context, client client2.AgentClient, log log.Logger) (*config2.Result, error) {
	// compress info
	workspaceInfo, err := client.AgentInfo()
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
	if cmd.ForceBuild {
		command += " --force-build"
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
	workspaceConfig := client.WorkspaceConfig()
	agentConfig := client.AgentConfig()

	// create container etc.
	result, err := agent.RunTunnelServer(
		cancelCtx,
		stdoutReader,
		stdinWriter,
		false,
		string(agentConfig.InjectGitCredentials) == "true" && workspaceConfig.Source.GitRepository != "",
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
