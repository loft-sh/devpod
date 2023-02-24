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
	"github.com/loft-sh/devpod/pkg/ide/goland"
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
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
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

			client, err := workspace2.ResolveWorkspace(ctx, devPodConfig, &provider2.WorkspaceIDEConfig{
				IDE: ide,
			}, args, cmd.ID, cmd.Provider, log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, ide, client)
		},
	}

	upCmd.Flags().StringVar(&cmd.ID, "id", "", "The id to use for the workspace")
	upCmd.Flags().StringVar(&cmd.IDE, "ide", "", "The IDE to open the workspace in. If empty will use vscode locally or in browser")
	return upCmd
}

// Run runs the command logic
func (cmd *UpCmd) Run(ctx context.Context, ide provider2.IDE, client client2.WorkspaceClient) error {
	// run devpod agent up
	result, err := devPodUp(ctx, client, log.Default)
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
	switch ide {
	case provider2.IDEVSCode:
		return startVSCodeLocally(client, log.Default)
	case provider2.IDEOpenVSCode:
		return startInBrowser(ctx, client, user, log.Default)
	case provider2.IDEGoland:
		return startGoland(result, client, log.Default)
	}

	return nil
}

func startGoland(result *config2.Result, client client2.WorkspaceClient, log log.Logger) error {
	log.Infof("Starting JetBrains Gateway...")
	remoteUser := config2.GetRemoteUser(result)
	err := open.Start(`jetbrains-gateway://connect#idePath=` + url.QueryEscape(goland.GetGolandDirectory(path.Join("/", "home", remoteUser))) + `&projectPath=` + url.QueryEscape(result.SubstitutionContext.ContainerWorkspaceFolder) + `&host=` + client.Workspace() + `.devpod&port=22&user=` + remoteUser + `&type=ssh&deploy=false`)
	if err != nil {
		return err
	}
	return nil
}

func startVSCodeLocally(client client2.WorkspaceClient, log log.Logger) error {
	log.Infof("Starting VSCode...")
	err := exec.Command("code", "--folder-uri", fmt.Sprintf("vscode-remote://ssh-remote+%s.devpod/workspaces/%s", client.Workspace(), client.Workspace())).Run()
	if err != nil {
		return err
	}

	return nil
}

func startInBrowser(ctx context.Context, client client2.WorkspaceClient, user string, log log.Logger) error {
	agentClient, ok := client.(client2.AgentClient)
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

func devPodUp(ctx context.Context, client client2.WorkspaceClient, log log.Logger) (*config2.Result, error) {
	err := startWait(ctx, client, true, log)
	if err != nil {
		return nil, err
	}

	agentClient, ok := client.(client2.AgentClient)
	if ok {
		return devPodUpServer(ctx, agentClient, log)
	}

	return nil, nil
}

func devPodUpServer(ctx context.Context, client client2.AgentClient, log log.Logger) (*config2.Result, error) {
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

		writer := log.ErrorStreamOnly().Writer(logrus.DebugLevel, false)
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

	// create container etc.
	result, err := agent.RunTunnelServer(cancelCtx, stdoutReader, stdinWriter, false, false, client.WorkspaceConfig(), log)
	if err != nil {
		return nil, errors.Wrap(err, "run tunnel server")
	}

	// wait until command finished
	return result, <-errChan
}
