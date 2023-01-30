package cmd

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider/types"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/template"
	"github.com/loft-sh/devpod/pkg/token"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/devpod/scripts"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"time"
)

var waitForInstanceConnectionTimeout = time.Minute * 5

// SSHCmd holds the ssh cmd flags
type SSHCmd struct {
	Stdio         bool
	JumpContainer bool

	Configure bool
	ID        string

	ShowAgentCommand bool
}

// NewSSHCmd creates a new ssh command
func NewSSHCmd() *cobra.Command {
	cmd := &SSHCmd{}
	sshCmd := &cobra.Command{
		Use:   "ssh",
		Short: "Starts a new ssh session to a workspace",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			workspace, provider, err := workspace2.GetWorkspace([]string{cmd.ID}, log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(context.Background(), workspace, provider)
		},
	}

	sshCmd.Flags().StringVar(&cmd.ID, "id", "", "The id of the workspace to use")
	sshCmd.Flags().BoolVar(&cmd.Configure, "configure", false, "If true will configure ssh for the given workspace")
	sshCmd.Flags().BoolVar(&cmd.Stdio, "stdio", false, "If true will tunnel connection through stdout and stdin")
	sshCmd.Flags().BoolVar(&cmd.ShowAgentCommand, "show-agent-command", false, "If true will show with which flags the agent needs to be executed remotely")
	_ = sshCmd.MarkFlagRequired("id")
	return sshCmd
}

// Run runs the command logic
func (cmd *SSHCmd) Run(ctx context.Context, workspace *config.Workspace, provider types.Provider) error {
	if cmd.ShowAgentCommand {
		return cmd.showAgentCommand(cmd.ID)
	}
	if cmd.Configure {
		return configureSSH(cmd.ID, "root")
	}

	err := startWait(ctx, provider, workspace, false, log.Default)
	if err != nil {
		return err
	}

	if cmd.Stdio {
		return jumpContainer(ctx, provider, workspace)
	}

	// TODO: Implement regular ssh client here
	return nil
}

func waitForInstanceConnection(ctx context.Context, provider types.ServerProvider, workspace *config.Workspace, log log.Logger) error {
	// do a simple hello world to check if we can get something
	startWaiting := time.Now()
	now := startWaiting
	for {
		reader := &bytes.Buffer{}
		cancelCtx, cancel := context.WithTimeout(ctx, time.Second*10)
		err := provider.RunCommand(cancelCtx, workspace, types.RunCommandOptions{
			Command: "echo -n devpod",
			Stdout:  reader,
		})
		cancel()
		if err != nil || reader.String() != "devpod" {
			if time.Since(now) > waitForInstanceConnectionTimeout {
				return errors.Wrap(err, "timeout waiting for instance connection")
			}

			time.Sleep(time.Second)
			if time.Since(startWaiting) > time.Second*10 {
				log.Infof("Waiting for devpod agent to come up...")
				startWaiting = time.Now()
			}
			continue
		}

		// run the actual command
		return nil
	}
}

func startWait(ctx context.Context, provider types.Provider, workspace *config.Workspace, create bool, log log.Logger) error {
	workspaceProvider, ok := provider.(types.WorkspaceProvider)
	if ok {
		err := startWaitWorkspace(ctx, workspaceProvider, workspace, create, log)
		if err != nil {
			return err
		}
	}

	serverProvider, ok := provider.(types.ServerProvider)
	if ok {
		err := startWaitServer(ctx, serverProvider, workspace, create, log)
		if err != nil {
			return err
		}
	}

	return nil
}

func startWaitWorkspace(ctx context.Context, provider types.WorkspaceProvider, workspace *config.Workspace, create bool, log log.Logger) error {
	startWaiting := time.Now()
	for {
		instanceStatus, err := provider.WorkspaceStatus(ctx, workspace, types.WorkspaceStatusOptions{})
		if err != nil {
			return errors.Wrap(err, "get instance status")
		} else if instanceStatus == types.StatusBusy {
			if time.Since(startWaiting) > time.Second*10 {
				log.Infof("Waiting for instance to come up...")
				startWaiting = time.Now()
			}

			time.Sleep(time.Second)
			continue
		} else if instanceStatus == types.StatusStopped {
			err = provider.WorkspaceStart(ctx, workspace, types.WorkspaceStartOptions{})
			if err != nil {
				return errors.Wrap(err, "start instance")
			}
		} else if instanceStatus == types.StatusNotFound {
			if create {
				// create environment
				err = provider.WorkspaceCreate(ctx, workspace, types.WorkspaceCreateOptions{})
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

func startWaitServer(ctx context.Context, provider types.ServerProvider, workspace *config.Workspace, create bool, log log.Logger) error {
	startWaiting := time.Now()
	for {
		instanceStatus, err := provider.Status(ctx, workspace, types.StatusOptions{})
		if err != nil {
			return errors.Wrap(err, "get instance status")
		} else if instanceStatus == types.StatusBusy {
			if time.Since(startWaiting) > time.Second*10 {
				log.Infof("Waiting for instance to come up...")
				startWaiting = time.Now()
			}

			time.Sleep(time.Second)
			continue
		} else if instanceStatus == types.StatusStopped {
			err = provider.Start(ctx, workspace, types.StartOptions{})
			if err != nil {
				return errors.Wrap(err, "start instance")
			}
		} else if instanceStatus == types.StatusNotFound {
			if create {
				// create environment
				err = provider.Create(ctx, workspace, types.CreateOptions{})
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("instance wasn't found")
			}
		}

		return waitForInstanceConnection(ctx, provider, workspace, log)
	}
}

func jumpContainer(ctx context.Context, provider types.Provider, workspace *config.Workspace) error {
	// get token
	tok, err := token.GenerateWorkspaceToken(workspace.ID)
	if err != nil {
		return err
	}

	workspaceProvider, ok := provider.(types.WorkspaceProvider)
	if ok {
		return jumpContainerWorkspace(ctx, workspaceProvider, workspace, tok)
	}

	serverProvider, ok := provider.(types.ServerProvider)
	if ok {
		return jumpContainerServer(ctx, serverProvider, workspace, tok)
	}

	return nil
}

func jumpContainerWorkspace(ctx context.Context, provider types.WorkspaceProvider, workspace *config.Workspace, tok string) error {
	err := provider.WorkspaceTunnel(ctx, workspace, types.WorkspaceTunnelOptions{
		Token:  tok,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	if err != nil {
		return err
	}

	return nil
}

func jumpContainerServer(ctx context.Context, provider types.ServerProvider, workspace *config.Workspace, tok string) error {
	// install devpod into the ssh machine
	t, err := template.FillTemplate(scripts.InstallDevPodTemplate, map[string]string{
		"BaseUrl": agent.DefaultAgentDownloadURL,
		"Command": fmt.Sprintf("sudo %s agent container-tunnel --id %s --token %s", agent.RemoteDevPodHelperLocation, workspace.ID, tok),
	})
	if err != nil {
		return err
	}

	// tunnel to container
	err = provider.RunCommand(ctx, workspace, types.RunCommandOptions{
		Command: t,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	})
	if err != nil {
		return errors.Wrap(err, "start tunnel")
	}

	return nil
}

func (cmd *SSHCmd) showAgentCommand(workspaceID string) error {
	t, _ := token.GenerateWorkspaceToken(workspaceID)
	fmt.Println(fmt.Sprintf("devpod agent ssh-server --token %s", t))
	return nil
}

func configureSSH(id, user string) error {
	err := devssh.ConfigureSSHConfig(id, user, log.Default)
	if err != nil {
		return err
	}

	return nil
}
