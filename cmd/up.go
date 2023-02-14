package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/open"
	"github.com/loft-sh/devpod/pkg/port"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/tunnel"
	"github.com/loft-sh/devpod/pkg/vscode"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"os/exec"
)

// UpCmd holds the up cmd flags
type UpCmd struct {
	flags.GlobalFlags

	ID      string
	Browser bool
}

// NewUpCmd creates a new up command
func NewUpCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &UpCmd{
		GlobalFlags: *flags,
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

			workspace, provider, err := workspace2.ResolveWorkspace(ctx, devPodConfig, args, cmd.ID, log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, workspace, provider)
		},
	}

	upCmd.Flags().StringVar(&cmd.ID, "id", "", "The id to use for the workspace")
	upCmd.Flags().BoolVar(&cmd.Browser, "browser", false, "If true will start VSCode in a browser")
	return upCmd
}

// Run runs the command logic
func (cmd *UpCmd) Run(ctx context.Context, workspace *provider2.Workspace, provider provider2.Provider) error {
	// run devpod agent up
	err := devPodUp(ctx, provider, workspace, log.Default)
	if err != nil {
		return err
	}

	// configure container ssh
	err = configureSSH(workspace.Context, workspace.ID, "vscode")
	if err != nil {
		return err
	}
	log.Default.Infof("Run 'ssh %s.devpod' to ssh into the devcontainer", workspace.ID)

	// start VSCode
	if cmd.Browser {
		return startInBrowser(ctx, workspace, provider, log.Default)
	} else {
		log.Default.Infof("Starting VSCode...")
		err = exec.Command("code", "--folder-uri", fmt.Sprintf("vscode-remote://ssh-remote+%s.devpod/workspaces/%s", workspace.ID, workspace.ID)).Run()
		if err != nil {
			return err
		}
	}

	return nil
}

func startInBrowser(ctx context.Context, workspace *provider2.Workspace, provider provider2.Provider, log log.Logger) error {
	serverProvider, ok := provider.(provider2.ServerProvider)
	if !ok {
		return fmt.Errorf("--browser is currently only supported for server providers")
	}

	// determine port
	vscodePort, err := port.FindAvailablePort(vscode.DefaultVSCodePort)
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

		// forward port
		forwardErr := make(chan error, 1)
		go func() {
			forwardErr <- devssh.PortForward(client, fmt.Sprintf("localhost:%d", vscodePort), fmt.Sprintf("localhost:%d", vscode.DefaultVSCodePort), log)
		}()

		// start openvscode
		command := fmt.Sprintf("%s agent openvscode --port %d", agent.RemoteDevPodHelperLocation, vscode.DefaultVSCodePort)
		log.Debugf("Running %s in container", command)
		err = devssh.Run(client, command, nil, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func devPodUp(ctx context.Context, provider provider2.Provider, workspace *provider2.Workspace, log log.Logger) error {
	serverProvider, ok := provider.(provider2.ServerProvider)
	if ok {
		return devPodUpServer(ctx, serverProvider, workspace, log)
	}

	workspaceProvider, ok := provider.(provider2.WorkspaceProvider)
	if ok {
		return startWaitWorkspace(ctx, workspaceProvider, workspace, true, log)
	}

	return nil
}

func devPodUpServer(ctx context.Context, provider provider2.ServerProvider, workspace *provider2.Workspace, log log.Logger) error {
	agentExists, err := startWaitServer(ctx, provider, workspace, true, log)
	if err != nil {
		return err
	}

	// inject agent
	if !agentExists {
		err = injectAgent(ctx, workspace.Provider.Agent.Path, workspace.Provider.Agent.DownloadURL, provider, workspace)
		if err != nil {
			return err
		}
	}

	// create pipes
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return err
	}

	// start server on stdio
	go func() {
		err := agent.StartTunnelServer(stdoutReader, stdinWriter, false, workspace, log)
		if err != nil {
			log.Errorf("Start tunnel server: %v", err)
		}
	}()

	// compress info
	workspaceInfo, err := provider2.NewAgentWorkspaceInfo(workspace)
	if err != nil {
		return err
	}

	// create container etc.
	log.Infof("Creating devcontainer...")
	err = provider.Command(ctx, workspace, provider2.CommandOptions{
		Command: fmt.Sprintf("%s agent up --workspace-info '%s'", workspace.Provider.Agent.Path, workspaceInfo),
		Stdin:   stdinReader,
		Stdout:  stdoutWriter,
		Stderr:  os.Stderr,
	})
	if err != nil {
		return err
	}

	return nil
}

func injectAgent(ctx context.Context, agentPath, agentURL string, provider provider2.ServerProvider, workspace *provider2.Workspace) error {
	// install devpod into the target
	err := agent.InjectAgent(func(command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		return provider.Command(ctx, workspace, provider2.CommandOptions{
			Command: command,
			Stdin:   stdin,
			Stdout:  stdout,
			Stderr:  stderr,
		})
	}, agentPath, agentURL, true)
	if err != nil {
		return err
	}

	return nil
}
