package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/config"
	config2 "github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/ide/openvscode"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/open"
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
	"strings"
	"time"
)

// UpCmd holds the up cmd flags
type UpCmd struct {
	*flags.GlobalFlags

	ID string

	IDE string
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

			if cmd.IDE == "" {
				cmd.IDE = string(ide.Detect())
			}
			ide, err := ide.Parse(cmd.IDE)
			if err != nil {
				return err
			}

			workspace, provider, err := workspace2.ResolveWorkspace(ctx, devPodConfig, &provider2.WorkspaceIDEConfig{
				IDE: ide,
			}, args, cmd.ID, log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, ide, workspace, provider)
		},
	}

	upCmd.Flags().StringVar(&cmd.ID, "id", "", "The id to use for the workspace")
	upCmd.Flags().StringVar(&cmd.IDE, "ide", "", "The IDE to open the workspace in. If empty will use vscode locally or in browser")
	return upCmd
}

// Run runs the command logic
func (cmd *UpCmd) Run(ctx context.Context, ide provider2.IDE, workspace *provider2.Workspace, provider provider2.Provider) error {
	// run devpod agent up
	result, err := devPodUp(ctx, provider, workspace, log.Default)
	if err != nil {
		return err
	}

	// get user from result
	user := config2.GetRemoteUser(result)

	// configure container ssh
	err = configureSSH(workspace.Context, workspace.ID, user)
	if err != nil {
		return err
	}
	log.Default.Infof("Run 'ssh %s.devpod' to ssh into the devcontainer", workspace.ID)

	// open ide
	switch ide {
	case provider2.IDEVSCode:
		return startLocally(workspace, log.Default)
	case provider2.IDEOpenVSCode:
		return startInBrowser(ctx, workspace, provider, log.Default)
	}

	return nil
}

func startLocally(workspace *provider2.Workspace, log log.Logger) error {
	log.Infof("Starting VSCode...")
	err := exec.Command("code", "--folder-uri", fmt.Sprintf("vscode-remote://ssh-remote+%s.devpod/workspaces/%s", workspace.ID, workspace.ID)).Run()
	if err != nil {
		return err
	}

	return nil
}

func startInBrowser(ctx context.Context, workspace *provider2.Workspace, provider provider2.Provider, log log.Logger) error {
	serverProvider, ok := provider.(provider2.ServerProvider)
	if !ok {
		return fmt.Errorf("--browser is currently only supported for server providers")
	}

	// determine port
	vscodePort, err := port.FindAvailablePort(openvscode.DefaultVSCodePort)
	if err != nil {
		return err
	}

	// wait until reachable then open browser
	go func() {
		err = open.Open(ctx, fmt.Sprintf("http://localhost:%d/?folder=/workspaces/%s", vscodePort, workspace.ID), log)
		if err != nil {
			log.Errorf("error opening vscode: %v", err)
		}

		log.Infof("Successfully started vscode in browser mode. Please keep this terminal open as long as you use VSCode browser version")
	}()

	// start in browser
	log.Infof("Starting vscode in browser mode...")
	err = tunnel.NewContainerTunnel(serverProvider, workspace, log).Run(ctx, nil, func(client *ssh.Client) error {
		log.Debugf("Connected to container")

		return devssh.PortForward(client, fmt.Sprintf("localhost:%d", vscodePort), fmt.Sprintf("localhost:%d", openvscode.DefaultVSCodePort), log)
	})
	if err != nil {
		return err
	}

	return nil
}

func devPodUp(ctx context.Context, provider provider2.Provider, workspace *provider2.Workspace, log log.Logger) (*config2.Result, error) {
	serverProvider, ok := provider.(provider2.ServerProvider)
	if ok {
		return devPodUpServer(ctx, serverProvider, workspace, log)
	}

	workspaceProvider, ok := provider.(provider2.WorkspaceProvider)
	if ok {
		err := startWaitWorkspace(ctx, workspaceProvider, workspace, true, log)
		return nil, err
	}

	return nil, nil
}

func devPodUpServer(ctx context.Context, provider provider2.ServerProvider, workspace *provider2.Workspace, log log.Logger) (*config2.Result, error) {
	err := startWaitServer(ctx, provider, workspace, true, log)
	if err != nil {
		return nil, err
	}

	// inject agent
	err = injectAgent(ctx, workspace.Provider.Agent.Path, workspace.Provider.Agent.DownloadURL, provider, workspace, log)
	if err != nil {
		return nil, err
	}

	// compress info
	workspaceInfo, err := provider2.NewAgentWorkspaceInfo(workspace)
	if err != nil {
		return nil, err
	}

	// create container etc.
	log.Infof("Creating devcontainer...")
	defer log.Debugf("Done creating devcontainer")
	command := fmt.Sprintf("%s agent workspace up --workspace-info '%s'", workspace.Provider.Agent.Path, workspaceInfo)
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

	// start server on stdio
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		defer log.Debugf("Done executing up command")
		defer cancel()

		writer := log.ErrorStreamOnly().Writer(logrus.ErrorLevel, false)
		defer writer.Close()

		errChan <- provider.Command(cancelCtx, workspace, provider2.CommandOptions{
			Command: command,
			Stdin:   stdinReader,
			Stdout:  stdoutWriter,
			Stderr:  writer,
		})
	}()

	// create container etc.
	result, err := agent.RunTunnelServer(cancelCtx, stdoutReader, stdinWriter, false, workspace, log)
	if err != nil {
		return nil, errors.Wrap(err, "run tunnel server")
	}

	// wait until command finished
	return result, <-errChan
}

func injectAgent(ctx context.Context, agentPath, agentURL string, provider provider2.ServerProvider, workspace *provider2.Workspace, log log.Logger) error {
	// install devpod into the target
	// do a simple hello world to check if we can get something
	startWaiting := time.Now()
	now := startWaiting
	for {
		err := agent.InjectAgent(func(command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			return provider.Command(ctx, workspace, provider2.CommandOptions{
				Command: command,
				Stdin:   stdin,
				Stdout:  stdout,
				Stderr:  stderr,
			})
		}, agentPath, agentURL, true, time.Second*10)
		if err != nil {
			if time.Since(now) > waitForInstanceConnectionTimeout {
				return errors.Wrap(err, "timeout waiting for instance connection")
			} else if strings.HasPrefix(err.Error(), "unexpected start line: ") || err == context.DeadlineExceeded {
				log.Infof("Waiting for devpod agent to come up...")
				startWaiting = time.Now()
				continue
			}

			return err
		}

		break
	}

	return nil
}
