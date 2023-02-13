package cmd

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/token"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"time"
)

var waitForInstanceConnectionTimeout = time.Minute * 5

// SSHCmd holds the ssh cmd flags
type SSHCmd struct {
	flags.GlobalFlags

	Stdio         bool
	JumpContainer bool

	Self      bool
	Configure bool
	ID        string
}

// NewSSHCmd creates a new ssh command
func NewSSHCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &SSHCmd{
		GlobalFlags: *flags,
	}
	sshCmd := &cobra.Command{
		Use:   "ssh",
		Short: "Starts a new ssh session to a workspace",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()
			devPodConfig, err := config.LoadConfig(cmd.Context)
			if err != nil {
				return err
			}

			var (
				workspace *provider2.Workspace
				provider  provider2.Provider
			)
			if cmd.Self {
				workspace, provider, err = workspace2.ResolveWorkspace(ctx, devPodConfig, []string{"."}, cmd.ID, log.Default)
				if err != nil {
					return err
				}
			} else {
				workspace, provider, err = workspace2.GetWorkspace(ctx, devPodConfig, []string{cmd.ID}, log.Default)
				if err != nil {
					return err
				}
			}

			return cmd.Run(ctx, workspace, provider)
		},
	}

	sshCmd.Flags().StringVar(&cmd.ID, "id", "", "The id of the workspace to use")
	sshCmd.Flags().BoolVar(&cmd.Configure, "configure", false, "If true will configure ssh for the given workspace")
	sshCmd.Flags().BoolVar(&cmd.Stdio, "stdio", false, "If true will tunnel connection through stdout and stdin")
	sshCmd.Flags().BoolVar(&cmd.Self, "self", false, "For testing only")
	_ = sshCmd.MarkFlagRequired("id")
	_ = sshCmd.Flags().MarkHidden("self")
	return sshCmd
}

// Run runs the command logic
func (cmd *SSHCmd) Run(ctx context.Context, workspace *provider2.Workspace, provider provider2.Provider) error {
	if cmd.Configure {
		return configureSSH(workspace.Context, cmd.ID, "root")
	}
	if cmd.Self {
		return configureSSHSelf(workspace, log.Default)
	}

	if cmd.Stdio {
		return jumpContainer(ctx, provider, workspace, log.Default)
	}

	// TODO: Implement regular ssh client here
	return nil
}

func waitForInstanceConnection(ctx context.Context, provider provider2.ServerProvider, workspace *provider2.Workspace, log log.Logger) (bool, error) {
	// get agent config
	agentConfig, err := provider.AgentConfig()
	if err != nil {
		return false, errors.Wrap(err, "get agent config")
	}
	agentPath := agentConfig.Path
	if agentPath == "" {
		agentPath = agent.RemoteDevPodHelperLocation
	}

	// do a simple hello world to check if we can get something
	startWaiting := time.Now()
	now := startWaiting
	for {
		reader := &bytes.Buffer{}
		cancelCtx, cancel := context.WithTimeout(ctx, time.Second*10)
		err := provider.Command(cancelCtx, workspace, provider2.CommandOptions{
			Command: fmt.Sprintf("%s version > /dev/null 2>&1 && echo -n exists || echo -n notexists", agentPath),
			Stdout:  reader,
		})
		cancel()
		if err != nil || (reader.String() != "exists" && reader.String() != "notexists") {
			if time.Since(now) > waitForInstanceConnectionTimeout {
				return false, errors.Wrap(err, "timeout waiting for instance connection")
			}

			time.Sleep(time.Second)
			if time.Since(startWaiting) > time.Second*10 {
				log.Infof("Waiting for devpod agent to come up...")
				startWaiting = time.Now()
			}
			continue
		}

		return reader.String() == "exists", nil
	}
}

func startWaitWorkspace(ctx context.Context, provider provider2.WorkspaceProvider, workspace *provider2.Workspace, create bool, log log.Logger) error {
	startWaiting := time.Now()
	for {
		instanceStatus, err := provider.Status(ctx, workspace, provider2.WorkspaceStatusOptions{})
		if err != nil {
			return errors.Wrap(err, "get instance status")
		} else if instanceStatus == provider2.StatusBusy {
			if time.Since(startWaiting) > time.Second*10 {
				log.Infof("Waiting for instance to come up...")
				startWaiting = time.Now()
			}

			time.Sleep(time.Second)
			continue
		} else if instanceStatus == provider2.StatusStopped {
			err = provider.Start(ctx, workspace, provider2.WorkspaceStartOptions{})
			if err != nil {
				return errors.Wrap(err, "start instance")
			}
		} else if instanceStatus == provider2.StatusNotFound {
			if create {
				// create environment
				err = provider.Create(ctx, workspace, provider2.WorkspaceCreateOptions{})
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("instance wasn't found")
			}
		}

		return nil
	}
}

func startWaitServer(ctx context.Context, provider provider2.ServerProvider, workspace *provider2.Workspace, create bool, log log.Logger) (bool, error) {
	startWaiting := time.Now()
	for {
		instanceStatus, err := provider.Status(ctx, workspace, provider2.StatusOptions{})
		if err != nil {
			return false, errors.Wrap(err, "get instance status")
		} else if instanceStatus == provider2.StatusBusy {
			if time.Since(startWaiting) > time.Second*10 {
				log.Infof("Waiting for instance to come up...")
				log.Debugf("Got status %s, expected: Running", instanceStatus)
				startWaiting = time.Now()
			}

			time.Sleep(time.Second)
			continue
		} else if instanceStatus == provider2.StatusStopped {
			err = provider.Start(ctx, workspace, provider2.StartOptions{})
			if err != nil {
				return false, errors.Wrap(err, "start instance")
			}
		} else if instanceStatus == provider2.StatusNotFound {
			if create {
				// create environment
				err = provider.Create(ctx, workspace, provider2.CreateOptions{})
				if err != nil {
					return false, err
				}
			} else {
				return false, fmt.Errorf("instance wasn't found")
			}
		}

		return waitForInstanceConnection(ctx, provider, workspace, log)
	}
}

func jumpContainer(ctx context.Context, provider provider2.Provider, workspace *provider2.Workspace, log log.Logger) error {
	workspaceProvider, ok := provider.(provider2.WorkspaceProvider)
	if ok {
		return jumpContainerWorkspace(ctx, workspaceProvider, workspace)
	}

	serverProvider, ok := provider.(provider2.ServerProvider)
	if ok {
		return jumpContainerServer(ctx, serverProvider, workspace, log)
	}

	return nil
}

func jumpContainerWorkspace(ctx context.Context, provider provider2.WorkspaceProvider, workspace *provider2.Workspace) error {
	err := startWaitWorkspace(ctx, provider, workspace, false, log.Default)
	if err != nil {
		return err
	}

	err = provider.Tunnel(ctx, workspace, provider2.WorkspaceTunnelOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	if err != nil {
		return err
	}

	return nil
}

func jumpContainerServer(ctx context.Context, provider provider2.ServerProvider, workspace *provider2.Workspace, log log.Logger) error {
	agentExists, err := startWaitServer(ctx, provider, workspace, false, log)
	if err != nil {
		return err
	}

	// get agent config
	agentConfig, err := agent.GetAgentConfig(provider)
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

	// tunnel to container
	return tunnelToContainer(ctx, provider, workspace, agentConfig)
}

func tunnelToContainer(ctx context.Context, provider provider2.ServerProvider, workspace *provider2.Workspace, agentConfig *provider2.ProviderAgentConfig) error {
	// get token
	tok, err := token.GenerateWorkspaceToken(workspace.Context, workspace.ID)
	if err != nil {
		return err
	}

	// compress info
	workspaceInfo, err := provider2.NewAgentWorkspaceInfo(workspace, provider)
	if err != nil {
		return err
	}

	// create readers
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return err
	}

	// tunnel to host
	// TODO: right now we have a tunnel in a tunnel, maybe its better to start 2 separate commands?
	tunnelChan := make(chan error, 1)
	go func() {
		tunnelChan <- provider.Command(ctx, workspace, provider2.CommandOptions{
			Command: fmt.Sprintf("%s helper ssh-server --token '%s' --stdio", agentConfig.Path, tok),
			Stdin:   stdinReader,
			Stdout:  stdoutWriter,
			Stderr:  os.Stderr,
		})
	}()

	// connect to container
	containerChan := make(chan error, 1)
	go func() {
		// connect via ssh over stdin / stdout
		keyBytes, err := devssh.GetPrivateKeyRaw(workspace.Context, workspace.ID)
		if err != nil {
			containerChan <- err
			return
		}

		// TODO: should we really exit here?
		sshClient, err := devssh.StdioClientFromKeyBytes(keyBytes, stdoutReader, stdinWriter, false)
		if err != nil {
			containerChan <- err
			return
		}
		defer sshClient.Close()

		// TODO: do port-forwarding etc. here with sshClient
		// go func() {}()

		// tunnel to container
		err = devssh.Run(sshClient, fmt.Sprintf("%s agent container-tunnel --token '%s' --workspace-info '%s'", agentConfig.Path, tok, workspaceInfo), os.Stdin, os.Stdout, os.Stderr)
		if err != nil {
			containerChan <- err
			return
		}
	}()

	// wait for result
	select {
	case err := <-containerChan:
		return errors.Wrap(err, "tunnel to container")
	case err := <-tunnelChan:
		return errors.Wrap(err, "connect to server")
	}
}

func configureSSHSelf(workspace *provider2.Workspace, log log.Logger) error {
	tok, err := token.GenerateWorkspaceToken(workspace.Context, workspace.ID)
	if err != nil {
		return err
	}

	err = devssh.ConfigureSSHConfigCommand(workspace.Context, workspace.ID, "", "devpod helper ssh-server --stdio --token "+tok, log)
	if err != nil {
		return err
	}

	err = exec.Command("code", "--folder-uri", fmt.Sprintf("vscode-remote://ssh-remote+%s.devpod/", workspace.ID)).Run()

	return nil
}

func configureSSH(context, workspaceID, user string) error {
	err := devssh.ConfigureSSHConfig(context, workspaceID, user, log.Default)
	if err != nil {
		return err
	}

	return nil
}
