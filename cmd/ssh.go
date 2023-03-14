package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/cmd/machine"
	"github.com/loft-sh/devpod/pkg/agent"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/token"
	"github.com/loft-sh/devpod/pkg/tunnel"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"time"
)

// SSHCmd holds the ssh cmd flags
type SSHCmd struct {
	*flags.GlobalFlags

	Stdio         bool
	JumpContainer bool

	Configure bool

	Command string
	User    string
}

// NewSSHCmd creates a new ssh command
func NewSSHCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &SSHCmd{
		GlobalFlags: flags,
	}
	sshCmd := &cobra.Command{
		Use:   "ssh",
		Short: "Starts a new ssh session to a workspace",
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()
			devPodConfig, err := config.LoadConfig(cmd.Context)
			if err != nil {
				return err
			}

			client, err := workspace2.GetWorkspace(devPodConfig, nil, args, true, log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, client)
		},
	}

	sshCmd.Flags().StringVar(&cmd.Command, "command", "", "The command to execute within the workspace")
	sshCmd.Flags().StringVar(&cmd.User, "user", "", "The user of the workspace to use")
	sshCmd.Flags().BoolVar(&cmd.Configure, "configure", false, "If true will configure ssh for the given workspace")
	sshCmd.Flags().BoolVar(&cmd.Stdio, "stdio", false, "If true will tunnel connection through stdout and stdin")
	_ = sshCmd.Flags().MarkHidden("self")
	return sshCmd
}

// Run runs the command logic
func (cmd *SSHCmd) Run(ctx context.Context, client client2.WorkspaceClient) error {
	if cmd.Configure {
		return configureSSH(client, "root")
	}

	return cmd.jumpContainer(ctx, client, log.Default.ErrorStreamOnly())
}

func startWait(ctx context.Context, client client2.WorkspaceClient, create bool, log log.Logger) error {
	startWaiting := time.Now()
	for {
		instanceStatus, err := client.Status(ctx, client2.StatusOptions{})
		if err != nil {
			return err
		} else if instanceStatus == client2.StatusBusy {
			if time.Since(startWaiting) > time.Second*10 {
				log.Infof("Waiting for instance to come up...")
				log.Debugf("Got status %s, expected: Running", instanceStatus)
				startWaiting = time.Now()
			}

			time.Sleep(time.Second)
			continue
		} else if instanceStatus == client2.StatusStopped {
			if create {
				// start environment
				err = client.Start(ctx, client2.StartOptions{})
				if err != nil {
					return errors.Wrap(err, "start instance")
				}
			} else {
				return fmt.Errorf("workspace is stopped")
			}
		} else if instanceStatus == client2.StatusNotFound {
			if create {
				// create environment
				err = client.Create(ctx, client2.CreateOptions{})
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("workspace wasn't found")
			}
		}

		return nil
	}
}

func (cmd *SSHCmd) jumpContainer(ctx context.Context, client client2.WorkspaceClient, log log.Logger) error {
	agentClient, ok := client.(client2.AgentClient)
	if ok {
		return cmd.jumpContainerServer(ctx, agentClient, log)
	}

	if client.ProviderType() == provider2.ProviderTypeDirect {
		return cmd.jumpContainerWorkspace(ctx, client)
	}

	return fmt.Errorf("unsupported workspace")
}

func (cmd *SSHCmd) jumpContainerWorkspace(ctx context.Context, client client2.WorkspaceClient) error {
	if !cmd.Stdio {
		return fmt.Errorf("unsupported")
	}

	err := startWait(ctx, client, false, log.Default)
	if err != nil {
		return err
	}

	err = client.Command(ctx, client2.CommandOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	if err != nil {
		return err
	}

	return nil
}

func (cmd *SSHCmd) jumpContainerServer(ctx context.Context, client client2.AgentClient, log log.Logger) error {
	err := startWait(ctx, client, false, log)
	if err != nil {
		return err
	}

	// get token
	tok, err := token.GetDevPodToken()
	if err != nil {
		return err
	}

	// compute workspace info
	workspaceInfo, err := client.AgentInfo()
	if err != nil {
		return err
	}

	// create credential helper in workspace
	var runInContainer tunnel.Handler
	if client.WorkspaceConfig().IDE.IDE != provider2.IDEVSCode && cmd.User != "" {
		runInContainer = func(client *ssh.Client) error {
			err := runCredentialsServer(ctx, client, cmd.User, log)
			if err != nil {
				log.Errorf("Error running credential server: %v", err)
			}

			<-ctx.Done()
			return nil
		}
	}

	// tunnel to container
	return tunnel.NewContainerTunnel(client, log).Run(ctx, func(sshClient *ssh.Client) error {
		writer := log.ErrorStreamOnly().Writer(logrus.InfoLevel, false)
		defer writer.Close()

		log.Debugf("Run outer container tunnel")
		command := fmt.Sprintf("%s agent container-tunnel --start-container --track-activity --token '%s' --workspace-info '%s'", client.AgentPath(), tok, workspaceInfo)
		if cmd.Debug {
			command += " --debug"
		}
		if cmd.User != "" {
			command += fmt.Sprintf(" --user='%s'", cmd.User)
		}
		if cmd.Stdio {
			return devssh.Run(sshClient, command, os.Stdin, os.Stdout, writer)
		}

		privateKey, err := devssh.GetDevPodPrivateKeyRaw()
		if err != nil {
			return err
		}

		return machine.StartSSHSession(ctx, privateKey, cmd.User, cmd.Command, func(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			return devssh.Run(sshClient, command, stdin, stdout, stderr)
		}, writer)
	}, runInContainer)
}

func runCredentialsServer(ctx context.Context, client *ssh.Client, user string, log log.Logger) error {
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer stdoutWriter.Close()

	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer stdinWriter.Close()

	// start server on stdio
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// only run credentials server if we know the user
	errChan := make(chan error, 1)
	go func() {
		defer cancel()
		writer := log.ErrorStreamOnly().Writer(logrus.DebugLevel, false)
		defer writer.Close()

		command := fmt.Sprintf("%s agent container credentials-server --user %s --configure-git-helper --configure-docker-helper", agent.RemoteDevPodHelperLocation, user)
		errChan <- devssh.Run(client, command, stdinReader, stdoutWriter, writer)
	}()

	_, err = agent.RunTunnelServer(cancelCtx, stdoutReader, stdinWriter, false, true, true, nil, log)
	if err != nil {
		return errors.Wrap(err, "run tunnel server")
	}

	// wait until command finished
	return <-errChan
}

func configureSSH(client client2.WorkspaceClient, user string) error {
	err := devssh.ConfigureSSHConfig(client.Context(), client.Workspace(), user, log.Default)
	if err != nil {
		return err
	}

	return nil
}
