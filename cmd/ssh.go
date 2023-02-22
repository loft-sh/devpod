package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/token"
	"github.com/loft-sh/devpod/pkg/tunnel"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"os"
	"os/exec"
	"time"
)

var waitForInstanceConnectionTimeout = time.Minute * 5

// SSHCmd holds the ssh cmd flags
type SSHCmd struct {
	*flags.GlobalFlags

	Stdio         bool
	JumpContainer bool

	Self      bool
	Configure bool
	ID        string
}

// NewSSHCmd creates a new ssh command
func NewSSHCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &SSHCmd{
		GlobalFlags: flags,
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
				workspace, provider, err = workspace2.ResolveWorkspace(ctx, devPodConfig, nil, []string{"."}, cmd.ID, log.Default)
				if err != nil {
					return err
				}
			} else {
				workspace, provider, err = workspace2.GetWorkspace(ctx, devPodConfig, nil, []string{cmd.ID}, log.Default)
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
		return jumpContainer(ctx, provider, workspace, log.Default.ErrorStreamOnly())
	}

	// TODO: Implement regular ssh client here
	return nil
}

func startWaitWorkspace(ctx context.Context, provider provider2.WorkspaceProvider, workspace *provider2.Workspace, create bool, log log.Logger) error {
	startWaiting := time.Now()
	for {
		instanceStatus, err := provider.Status(ctx, workspace, provider2.WorkspaceStatusOptions{})
		if err != nil {
			return err
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

func startWaitServer(ctx context.Context, provider provider2.ServerProvider, workspace *provider2.Workspace, create bool, log log.Logger) error {
	startWaiting := time.Now()
	for {
		instanceStatus, err := provider.Status(ctx, workspace, provider2.StatusOptions{})
		if err != nil {
			return err
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
				return errors.Wrap(err, "start instance")
			}
		} else if instanceStatus == provider2.StatusNotFound {
			if create {
				// create environment
				err = provider.Create(ctx, workspace, provider2.CreateOptions{})
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
	err := startWaitServer(ctx, provider, workspace, false, log)
	if err != nil {
		return err
	}

	// inject agent
	err = injectAgent(ctx, workspace.Provider.Agent.Path, workspace.Provider.Agent.DownloadURL, provider, workspace, log)
	if err != nil {
		return err
	}

	// get token
	tok, err := token.GenerateWorkspaceToken(workspace.Context, workspace.ID)
	if err != nil {
		return err
	}

	// compute workspace info
	workspaceInfo, err := provider2.NewAgentWorkspaceInfo(workspace)
	if err != nil {
		return err
	}

	// tunnel to container
	return tunnel.NewContainerTunnel(provider, workspace, log).Run(ctx, func(sshClient *ssh.Client) error {
		return devssh.Run(sshClient, fmt.Sprintf("%s agent container-tunnel --token '%s' --workspace-info '%s'", workspace.Provider.Agent.Path, tok, workspaceInfo), os.Stdin, os.Stdout, os.Stderr)
	}, nil)
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
	if err != nil {
		return err
	}
	return nil
}

func configureSSH(context, workspaceID, user string) error {
	err := devssh.ConfigureSSHConfig(context, workspaceID, user, log.Default)
	if err != nil {
		return err
	}

	return nil
}
