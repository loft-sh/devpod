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
	"github.com/loft-sh/devpod/pkg/port"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/tunnel"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/terminal"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

// SSHCmd holds the ssh cmd flags
type SSHCmd struct {
	*flags.GlobalFlags

	ForwardPortsTimeout string
	ForwardPorts        []string

	Stdio           bool
	JumpContainer   bool
	AgentForwarding bool

	StartServices bool

	Proxy bool

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

			client, err := workspace2.GetWorkspace(devPodConfig, args, true, log.Default.ErrorStreamOnly())
			if err != nil {
				return err
			}

			return cmd.Run(ctx, devPodConfig, client, log.Default.ErrorStreamOnly())
		},
	}

	sshCmd.Flags().StringArrayVarP(&cmd.ForwardPorts, "forward-ports", "L", []string{}, "Specifies that connections to the given TCP port or Unix socket on the local (client) host are to be forwarded to the given host and port, or Unix socket, on the remote side.")
	sshCmd.Flags().StringVar(&cmd.ForwardPortsTimeout, "forward-ports-timeout", "", "Specifies the timeout after which the command should terminate when the ports are unused.")
	sshCmd.Flags().StringVar(&cmd.Command, "command", "", "The command to execute within the workspace")
	sshCmd.Flags().StringVar(&cmd.User, "user", "", "The user of the workspace to use")
	sshCmd.Flags().BoolVar(&cmd.Proxy, "proxy", false, "If true will act as intermediate proxy for a proxy provider")
	sshCmd.Flags().BoolVar(&cmd.AgentForwarding, "agent-forwarding", true, "If true forward the local ssh keys to the remote machine")
	sshCmd.Flags().BoolVar(&cmd.Stdio, "stdio", false, "If true will tunnel connection through stdout and stdin")
	sshCmd.Flags().BoolVar(&cmd.StartServices, "start-services", true, "If false will not start any port-forwarding or git / docker credentials helper")
	return sshCmd
}

// Run runs the command logic
func (cmd *SSHCmd) Run(ctx context.Context, devPodConfig *config.Config, client client2.BaseWorkspaceClient, log log.Logger) error {
	// add ssh keys to agent
	if !cmd.Proxy && devPodConfig.ContextOption(config.ContextOptionSSHAgentForwarding) == "true" && devPodConfig.ContextOption(config.ContextOptionSSHAddPrivateKeys) == "true" {
		log.Debug("Adding ssh keys to agent, disable via 'devpod context set-options -o SSH_ADD_PRIVATE_KEYS=false'")
		err := devssh.AddPrivateKeysToAgent(ctx, log)
		if err != nil {
			log.Debugf("Error adding private keys to ssh-agent: %v", err)
		}
	}

	// get user
	if cmd.User == "" {
		var err error
		cmd.User, err = devssh.GetUser(client.Workspace())
		if err != nil {
			return err
		}
	}

	// check if regular workspace client
	workspaceClient, ok := client.(client2.WorkspaceClient)
	if ok {
		return cmd.jumpContainer(ctx, devPodConfig, workspaceClient, log)
	}

	// check if proxy client
	proxyClient, ok := client.(client2.ProxyClient)
	if ok {
		return cmd.startProxyTunnel(ctx, devPodConfig, proxyClient, log)
	}

	return nil
}

func (cmd *SSHCmd) startProxyTunnel(ctx context.Context, devPodConfig *config.Config, client client2.ProxyClient, log log.Logger) error {
	log.Debugf("Start proxy tunnel")
	return tunnel.NewTunnel(
		ctx,
		func(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
			return client.Ssh(ctx, client2.SshOptions{
				Stdin:  stdin,
				Stdout: stdout,
			})
		},
		func(ctx context.Context, containerClient *ssh.Client) error {
			return cmd.startTunnel(ctx, devPodConfig, containerClient, client.WorkspaceConfig().IDE.Name, log)
		},
	)
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
	// lock the workspace as long as we init the connection
	unlockOnce := sync.Once{}
	err := client.Lock(ctx)
	if err != nil {
		return err
	}
	defer unlockOnce.Do(client.Unlock)

	// start the workspace
	err = startWait(ctx, client, false, log)
	if err != nil {
		return err
	}

	// tunnel to container
	return tunnel.NewContainerTunnel(client, cmd.Proxy, log).Run(ctx, func(ctx context.Context, containerClient *ssh.Client) error {
		// we have a connection to the container, make sure others can connect as well
		unlockOnce.Do(client.Unlock)

		// start ssh tunnel
		return cmd.startTunnel(ctx, devPodConfig, containerClient, client.WorkspaceConfig().IDE.Name, log)
	})
}

func (cmd *SSHCmd) forwardPorts(ctx context.Context, containerClient *ssh.Client, log log.Logger) error {
	timeout := time.Duration(0)
	if cmd.ForwardPortsTimeout != "" {
		var err error
		timeout, err = time.ParseDuration(cmd.ForwardPortsTimeout)
		if err != nil {
			return fmt.Errorf("parse forward ports timeout: %w", err)
		}

		log.Infof("Using port forwarding timeout of %s", cmd.ForwardPortsTimeout)
	}

	errChan := make(chan error, len(cmd.ForwardPorts))
	for _, portMapping := range cmd.ForwardPorts {
		mapping, err := port.ParsePortSpec(portMapping)
		if err != nil {
			return fmt.Errorf("parse port mapping: %w", err)
		}

		// start the forwarding
		log.Infof("Forwarding local %s/%s to remote %s/%s", mapping.Host.Protocol, mapping.Host.Address, mapping.Container.Protocol, mapping.Container.Address)
		go func(portMapping string) {
			err := devssh.PortForward(ctx, containerClient, mapping.Host.Protocol, mapping.Host.Address, mapping.Container.Protocol, mapping.Container.Address, timeout, log)
			if err != nil {
				errChan <- fmt.Errorf("error forwarding %s: %w", portMapping, err)
			}
		}(portMapping)
	}

	return <-errChan
}

func (cmd *SSHCmd) startTunnel(ctx context.Context, devPodConfig *config.Config, containerClient *ssh.Client, ideName string, log log.Logger) error {
	// check if we should forward ports
	if len(cmd.ForwardPorts) > 0 {
		return cmd.forwardPorts(ctx, containerClient, log)
	}

	// start port-forwarding etc.
	if !cmd.Proxy && cmd.StartServices {
		go cmd.startServices(ctx, devPodConfig, containerClient, ideName, log)
	}

	// start ssh
	writer := log.ErrorStreamOnly().Writer(logrus.InfoLevel, false)
	defer writer.Close()

	log.Debugf("Run outer container tunnel")
	command := fmt.Sprintf("'%s' helper ssh-server --track-activity --stdio", agent.ContainerDevPodHelperLocation)
	if cmd.Debug {
		command += " --debug"
	}
	if cmd.User != "" && cmd.User != "root" {
		command = fmt.Sprintf("su -c \"%s\" '%s'", command, cmd.User)
	}
	if cmd.Proxy || cmd.Stdio {
		return devssh.Run(ctx, containerClient, command, os.Stdin, os.Stdout, writer)
	}

	return machine.StartSSHSession(ctx, cmd.User, cmd.Command, !cmd.Proxy && cmd.AgentForwarding && devPodConfig.ContextOption(config.ContextOptionSSHAgentForwarding) == "true", func(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		return devssh.Run(ctx, containerClient, command, stdin, stdout, stderr)
	}, writer)
}

func (cmd *SSHCmd) startServices(ctx context.Context, devPodConfig *config.Config, containerClient *ssh.Client, ideName string, log log.Logger) {
	if cmd.User != "" {
		gitCredentials := ideName != string(config.IDEVSCode)
		err := tunnel.RunInContainer(
			ctx,
			devPodConfig,
			containerClient,
			cmd.User,
			false,
			gitCredentials,
			true,
			nil,
			log,
		)
		if err != nil {
			log.Debugf("Error running credential server: %v", err)
		}
	}
}
