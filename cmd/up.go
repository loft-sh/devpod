package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io"
	"os"
	"os/exec"
)

// UpCmd holds the up cmd flags
type UpCmd struct {
	flags.GlobalFlags

	Snapshot bool
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

			workspace, provider, err := workspace2.ResolveWorkspace(ctx, devPodConfig, args, log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, workspace, provider)
		},
	}

	upCmd.Flags().BoolVar(&cmd.Snapshot, "snapshot", false, "If true will create a snapshot for the environment")
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
	log.Default.Infof("Starting VSCode...")
	err = exec.Command("code", "--folder-uri", fmt.Sprintf("vscode-remote://ssh-remote+%s.devpod/workspaces/%s", workspace.ID, workspace.ID)).Run()
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

	// get agent config
	agentConfig, err := getAgentConfig(provider)
	if err != nil {
		return err
	}

	// inject agent
	if !agentExists {
		err = injectAgent(ctx, agentConfig.Path, agentConfig.DownloadURL, provider, workspace)
		if err != nil {
			return err
		}
	}

	// create pipes
	stdoutReader, stdoutWriter := io.Pipe()
	stdinReader, stdinWriter := io.Pipe()

	// start server on stdio
	go func() {
		err := agent.StartTunnelServer(stdoutReader, stdinWriter, false, workspace, log)
		if err != nil {
			log.Errorf("Start tunnel server: %v", err)
		}
	}()

	// compress info
	workspaceInfo, err := agent.NewAgentWorkspaceInfo(workspace, provider)
	if err != nil {
		return err
	}

	// create container etc.
	log.Infof("Creating devcontainer...")
	err = provider.Command(ctx, workspace, provider2.CommandOptions{
		Command: fmt.Sprintf("sudo %s agent up --workspace-info '%s'", agentConfig.Path, workspaceInfo),
		Stdin:   stdinReader,
		Stdout:  stdoutWriter,
		Stderr:  os.Stderr,
	})
	if err != nil {
		return err
	}

	return nil
}

func getAgentConfig(provider provider2.Provider) (*provider2.ProviderAgentConfig, error) {
	agentConfig, err := provider.AgentConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get agent config")
	}
	if agentConfig.Path == "" {
		agentConfig.Path = agent.RemoteDevPodHelperLocation
	}
	if agentConfig.DownloadURL == "" {
		agentConfig.DownloadURL = agent.DefaultAgentDownloadURL
	}

	return agentConfig, nil
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
