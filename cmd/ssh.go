package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/cmd/machine"
	"github.com/loft-sh/devpod/pkg/agent"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/terminal"
	"github.com/loft-sh/devpod/pkg/token"
	"github.com/loft-sh/devpod/pkg/tunnel"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

// SSHCmd holds the ssh cmd flags
type SSHCmd struct {
	*flags.GlobalFlags

	Stdio           bool
	JumpContainer   bool
	AgentForwarding bool

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
			devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
			if err != nil {
				return err
			}

			client, err := workspace2.GetWorkspace(devPodConfig, args, true, log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, devPodConfig, client)
		},
	}

	sshCmd.Flags().StringVar(&cmd.Command, "command", "", "The command to execute within the workspace")
	sshCmd.Flags().StringVar(&cmd.User, "user", "", "The user of the workspace to use")
	sshCmd.Flags().BoolVar(&cmd.AgentForwarding, "agent-forwarding", true, "If true forward the local ssh keys to the remote machine")
	sshCmd.Flags().BoolVar(&cmd.Stdio, "stdio", false, "If true will tunnel connection through stdout and stdin")
	_ = sshCmd.Flags().MarkHidden("self")
	return sshCmd
}

// Run runs the command logic
func (cmd *SSHCmd) Run(ctx context.Context, devPodConfig *config.Config, client client2.WorkspaceClient) error {
	return cmd.jumpContainer(ctx, devPodConfig, client, log.Default.ErrorStreamOnly())
}

func startWait(ctx context.Context, client client2.WorkspaceClient, create bool, log log.Logger) error {
	startWaiting := time.Now()
	for {
		instanceStatus, err := client.Status(ctx, client2.StatusOptions{})
		if err != nil {
			return err
		} else if instanceStatus == client2.StatusBusy {
			if time.Since(startWaiting) > time.Second*10 {
				log.Infof("Waiting for workspace to come up...")
				log.Debugf("Got status %s, expected: Running", instanceStatus)
				startWaiting = time.Now()
			}

			time.Sleep(time.Second * 2)
			continue
		} else if instanceStatus == client2.StatusStopped {
			if create {
				// start environment
				err = client.Start(ctx, client2.StartOptions{})
				if err != nil {
					return errors.Wrap(err, "start workspace")
				}
			} else {
				if !terminal.IsTerminalIn {
					_ = beeep.Notify("DevPod Workspace is stopped", "DevPod Workspace is stopped, please restart the workspace", "assets/warning.png")
				}

				return fmt.Errorf("DevPod workspace is stopped")
			}
		} else if instanceStatus == client2.StatusNotFound {
			if create {
				// create environment
				err = client.Create(ctx, client2.CreateOptions{})
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("DevPod workspace wasn't found")
			}
		}

		return nil
	}
}

func (cmd *SSHCmd) jumpContainer(ctx context.Context, devPodConfig *config.Config, client client2.WorkspaceClient, log log.Logger) error {
	unlockOnce := sync.Once{}
	client.Lock()
	defer unlockOnce.Do(client.Unlock)

	// start the workspace
	err := startWait(ctx, client, false, log)
	if err != nil {
		return err
	}

	// get token
	tok, err := token.GetDevPodToken()
	if err != nil {
		return err
	}

	// get user
	if cmd.User == "" {
		cmd.User, err = devssh.GetUser(client.Workspace())
		if err != nil {
			return err
		}
	}

	// tunnel to container
	return tunnel.NewContainerTunnel(client, log).Run(
		ctx,
		func(ctx context.Context, hostClient, containerClient *ssh.Client) error {
			// we have a connection to the container, make sure others can connect as well
			unlockOnce.Do(client.Unlock)

			// start port-forwarding etc.
			go func() {
				if cmd.User != "" {
					gitCredentials := client.WorkspaceConfig().IDE.Name != string(config.IDEVSCode)
					err := tunnel.RunInContainer(
						ctx,
						client,
						devPodConfig,
						hostClient,
						containerClient,
						cmd.User,
						false,
						gitCredentials,
						true,
						nil,
						log,
					)
					if err != nil {
						log.Errorf("Error running credential server: %v", err)
					}
				}
			}()

			// start ssh
			writer := log.ErrorStreamOnly().Writer(logrus.InfoLevel, false)
			defer writer.Close()

			log.Debugf("Run outer container tunnel")
			command := fmt.Sprintf("'%s' helper ssh-server --track-activity --token '%s' --stdio", agent.ContainerDevPodHelperLocation, tok)
			if cmd.Debug {
				command += " --debug"
			}
			if cmd.User != "" {
				command = fmt.Sprintf("su -c \"%s\" '%s'", command, cmd.User)
			}
			if cmd.Stdio {
				return devssh.Run(ctx, containerClient, command, os.Stdin, os.Stdout, writer)
			}

			privateKey, err := devssh.GetDevPodPrivateKeyRaw()
			if err != nil {
				return err
			}

			return machine.StartSSHSession(ctx, privateKey, cmd.User, cmd.Command, cmd.AgentForwarding, func(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
				return devssh.Run(ctx, containerClient, command, stdin, stdout, stderr)
			}, writer)
		},
	)
}

func configureSSH(client client2.WorkspaceClient, user string) error {
	err := devssh.ConfigureSSHConfig(client.Context(), client.Workspace(), user, log.Default)
	if err != nil {
		return err
	}

	return nil
}
