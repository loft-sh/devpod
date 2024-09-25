package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/cmd/machine"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/agent/tunnelserver"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/gpg"
	"github.com/loft-sh/devpod/pkg/port"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/tunnel"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
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
	ReverseForwardPorts []string

	Stdio                     bool
	JumpContainer             bool
	AgentForwarding           bool
	GPGAgentForwarding        bool
	GitSSHSignatureForwarding bool

	StartServices bool

	Proxy bool

	Command string
	User    string
	WorkDir string
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
			devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
			if err != nil {
				return err
			}
			if err := mergeDevPodSshOptions(cmd); err != nil {
				return err
			}
			if cmd.Proxy {
				// merge context options from env
				config.MergeContextOptions(devPodConfig.Current(), os.Environ())
			}

			client, err := workspace2.GetWorkspace(devPodConfig, args, true, log.Default.ErrorStreamOnly())
			if err != nil {
				return err
			}

			ctx := context.Background()
			return cmd.Run(ctx, devPodConfig, client, log.Default.ErrorStreamOnly())
		},
	}

	sshCmd.Flags().StringArrayVarP(&cmd.ForwardPorts, "forward-ports", "L", []string{}, "Specifies that connections to the given TCP port or Unix socket on the local (client) host are to be forwarded to the given host and port, or Unix socket, on the remote side.")
	sshCmd.Flags().StringArrayVarP(&cmd.ReverseForwardPorts, "reverse-forward-ports", "R", []string{}, "Specifies that connections to the given TCP port or Unix socket on the local (client) host are to be reverse forwarded to the given host and port, or Unix socket, on the remote side.")
	sshCmd.Flags().StringVar(&cmd.ForwardPortsTimeout, "forward-ports-timeout", "", "Specifies the timeout after which the command should terminate when the ports are unused.")
	sshCmd.Flags().StringVar(&cmd.Command, "command", "", "The command to execute within the workspace")
	sshCmd.Flags().StringVar(&cmd.User, "user", "", "The user of the workspace to use")
	sshCmd.Flags().StringVar(&cmd.WorkDir, "workdir", "", "The working directory in the container")
	sshCmd.Flags().BoolVar(&cmd.Proxy, "proxy", false, "If true will act as intermediate proxy for a proxy provider")
	sshCmd.Flags().BoolVar(&cmd.AgentForwarding, "agent-forwarding", true, "If true forward the local ssh keys to the remote machine")
	sshCmd.Flags().BoolVar(&cmd.GPGAgentForwarding, "gpg-agent-forwarding", false, "If true forward the local gpg-agent to the remote machine")
	sshCmd.Flags().BoolVar(&cmd.Stdio, "stdio", false, "If true will tunnel connection through stdout and stdin")
	sshCmd.Flags().BoolVar(&cmd.StartServices, "start-services", true, "If false will not start any port-forwarding or git / docker credentials helper")
	return sshCmd
}

// Run runs the command logic
func (cmd *SSHCmd) Run(
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.BaseWorkspaceClient,
	log log.Logger) error {
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
		cmd.User, err = devssh.GetUser(client.WorkspaceConfig().ID, client.WorkspaceConfig().SSHConfigPath)
		if err != nil {
			return err
		}
	}

	// set default context if needed
	if cmd.Context == "" {
		cmd.Context = devPodConfig.DefaultContext
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

func (cmd *SSHCmd) startProxyTunnel(
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.ProxyClient,
	log log.Logger,
) error {
	log.Debugf("Start proxy tunnel")
	return tunnel.NewTunnel(
		ctx,
		func(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
			return client.Ssh(ctx, client2.SshOptions{
				User:   cmd.User,
				Stdin:  stdin,
				Stdout: stdout,
			})
		},
		func(ctx context.Context, containerClient *ssh.Client) error {
			return cmd.startTunnel(ctx, devPodConfig, containerClient, client.Workspace(), log)
		},
	)
}

func startWait(
	ctx context.Context,
	client client2.WorkspaceClient,
	create bool,
	log log.Logger,
) error {
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

func (cmd *SSHCmd) jumpContainer(
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.WorkspaceClient,
	log log.Logger,
) error {
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
	return tunnel.NewContainerTunnel(client, cmd.Proxy, log).
		Run(ctx, func(ctx context.Context, containerClient *ssh.Client) error {
			// we have a connection to the container, make sure others can connect as well
			unlockOnce.Do(client.Unlock)

			// start ssh tunnel
			return cmd.startTunnel(ctx, devPodConfig, containerClient, client.Workspace(), log)
		}, devPodConfig)
}

func (cmd *SSHCmd) forwardTimeout(log log.Logger) (time.Duration, error) {
	timeout := time.Duration(0)
	if cmd.ForwardPortsTimeout != "" {
		timeout, err := time.ParseDuration(cmd.ForwardPortsTimeout)
		if err != nil {
			return timeout, fmt.Errorf("parse forward ports timeout: %w", err)
		}

		log.Infof("Using port forwarding timeout of %s", cmd.ForwardPortsTimeout)
	}

	return timeout, nil
}

func (cmd *SSHCmd) reverseForwardPorts(
	ctx context.Context,
	containerClient *ssh.Client,
	log log.Logger,
) error {
	timeout, err := cmd.forwardTimeout(log)
	if err != nil {
		return fmt.Errorf("parse forward ports timeout: %w", err)
	}

	errChan := make(chan error, len(cmd.ReverseForwardPorts))
	for _, portMapping := range cmd.ReverseForwardPorts {
		mapping, err := port.ParsePortSpec(portMapping)
		if err != nil {
			return fmt.Errorf("parse port mapping: %w", err)
		}

		// start the forwarding
		log.Infof(
			"Reverse forwarding local %s/%s to remote %s/%s",
			mapping.Host.Protocol,
			mapping.Host.Address,
			mapping.Container.Protocol,
			mapping.Container.Address,
		)
		go func(portMapping string) {
			err := devssh.ReversePortForward(
				ctx,
				containerClient,
				mapping.Host.Protocol,
				mapping.Host.Address,
				mapping.Container.Protocol,
				mapping.Container.Address,
				timeout,
				log,
			)
			if err != nil {
				errChan <- fmt.Errorf("error forwarding %s: %w", portMapping, err)
			}
		}(portMapping)
	}

	return <-errChan
}

func (cmd *SSHCmd) forwardPorts(
	ctx context.Context,
	containerClient *ssh.Client,
	log log.Logger,
) error {
	timeout, err := cmd.forwardTimeout(log)
	if err != nil {
		return fmt.Errorf("parse forward ports timeout: %w", err)
	}

	errChan := make(chan error, len(cmd.ForwardPorts))
	for _, portMapping := range cmd.ForwardPorts {
		mapping, err := port.ParsePortSpec(portMapping)
		if err != nil {
			return fmt.Errorf("parse port mapping: %w", err)
		}

		// start the forwarding
		log.Infof(
			"Forwarding local %s/%s to remote %s/%s",
			mapping.Host.Protocol,
			mapping.Host.Address,
			mapping.Container.Protocol,
			mapping.Container.Address,
		)
		go func(portMapping string) {
			err := devssh.PortForward(
				ctx,
				containerClient,
				mapping.Host.Protocol,
				mapping.Host.Address,
				mapping.Container.Protocol,
				mapping.Container.Address,
				timeout,
				log,
			)
			if err != nil {
				errChan <- fmt.Errorf("error forwarding %s: %w", portMapping, err)
			}
		}(portMapping)
	}

	return <-errChan
}

func (cmd *SSHCmd) startTunnel(ctx context.Context, devPodConfig *config.Config, containerClient *ssh.Client, workspaceName string, log log.Logger) error {
	// check if we should forward ports
	if len(cmd.ForwardPorts) > 0 {
		return cmd.forwardPorts(ctx, containerClient, log)
	}

	// check if we should reverse forward ports
	if len(cmd.ReverseForwardPorts) > 0 && !cmd.GPGAgentForwarding {
		return cmd.reverseForwardPorts(ctx, containerClient, log)
	}

	// start port-forwarding etc.
	if !cmd.Proxy && cmd.StartServices {
		go cmd.startServices(ctx, devPodConfig, containerClient, cmd.GitUsername, cmd.GitToken, log)
	}

	// start ssh
	writer := log.ErrorStreamOnly().Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// check if we should do gpg agent forwarding
	if cmd.GPGAgentForwarding || devPodConfig.ContextOption(config.ContextOptionGPGAgentForwarding) == "true" {
		// Check if a forwarding is already enabled and running, in that case
		// we skip the forwarding and keep using the original one
		if gpg.IsGpgTunnelRunning(cmd.User, ctx, containerClient, log) {
			log.Debugf("gpg: exporting already running, skipping")
		} else {
			err := cmd.setupGPGAgent(ctx, containerClient, log)
			if err != nil {
				return err
			}
		}
	}

	workdir := filepath.Join("/workspaces", workspaceName)
	if cmd.WorkDir != "" {
		workdir = cmd.WorkDir
	}

	log.Debugf("Run outer container tunnel")
	command := fmt.Sprintf("'%s' helper ssh-server --track-activity --stdio --workdir '%s'", agent.ContainerDevPodHelperLocation, workdir)
	if cmd.Debug {
		command += " --debug"
	}
	if !cmd.Proxy && cmd.User != "" && cmd.User != "root" {
		command = fmt.Sprintf("su -c \"%s\" '%s'", command, cmd.User)
	}

	// Traffic is coming in from the outside, we need to forward it to the container
	if cmd.Proxy || cmd.Stdio {
		if cmd.Proxy {
			go func() {
				if err := cmd.startRunnerServices(ctx, devPodConfig, containerClient, log); err != nil {
					log.Error(err)
				}
			}()
		}

		return devssh.Run(ctx, containerClient, command, os.Stdin, os.Stdout, writer)
	}

	return machine.StartSSHSession(
		ctx,
		cmd.User,
		cmd.Command,
		!cmd.Proxy && cmd.AgentForwarding &&
			devPodConfig.ContextOption(config.ContextOptionSSHAgentForwarding) == "true",
		func(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			return devssh.Run(ctx, containerClient, command, stdin, stdout, stderr)
		},
		writer,
	)
}

func (cmd *SSHCmd) startServices(
	ctx context.Context,
	devPodConfig *config.Config,
	containerClient *ssh.Client,
	gitUsername,
	gitToken string,
	log log.Logger,
) {
	if cmd.User != "" {
		err := tunnel.RunInContainer(
			ctx,
			devPodConfig,
			containerClient,
			cmd.User,
			false,
			nil,
			gitUsername,
			gitToken,
			log,
		)
		if err != nil {
			log.Debugf("Error running credential server: %v", err)
		}
	}
}

func (cmd *SSHCmd) startRunnerServices(
	ctx context.Context,
	devPodConfig *config.Config,
	containerClient *ssh.Client,
	log log.Logger,
) error {
	// check prerequisites
	allowGitCredentials := devPodConfig.ContextOption(config.ContextOptionSSHInjectGitCredentials) == "true"
	allowDockerCredentials := devPodConfig.ContextOption(config.ContextOptionSSHInjectDockerCredentials) == "true"

	// prepare pipes
	stdoutReader, stdoutWriter, stdinReader, stdinWriter, err := preparePipes()
	if err != nil {
		return fmt.Errorf("prepare pipes: %w", err)
	}
	defer stdoutWriter.Close()
	defer stdinWriter.Close()

	// prepare context
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	errChan := make(chan error, 2)

	// start credentials server in workspace
	go func() {
		errChan <- startWorkspaceCredentialServer(cancelCtx, containerClient, cmd.User, allowGitCredentials, allowDockerCredentials, stdinReader, stdoutWriter, log)
	}()

	// start runner services server locally
	go func() {
		errChan <- startLocalServer(cancelCtx, allowGitCredentials, allowDockerCredentials, cmd.GitUsername, cmd.GitToken, stdoutReader, stdinWriter, log)
	}()

	return <-errChan
}

// setupGPGAgent will forward a local gpg-agent into the remote container
// this works by using cmd/agent/workspace/setup_gpg
func (cmd *SSHCmd) setupGPGAgent(
	ctx context.Context,
	containerClient *ssh.Client,
	log log.Logger,
) error {
	writer := log.ErrorStreamOnly().Writer(logrus.InfoLevel, false)
	defer writer.Close()

	log.Debugf("gpg: exporting gpg public key from host")

	// Read the user's public keys and ownertrust from GPG.
	// These commands are executed LOCALLY, the output will be imported by the remote gpg
	pubKeyExport, err := gpg.GetHostPubKey()
	if err != nil {
		return fmt.Errorf("export local public keys from GPG: %w", err)
	}

	log.Debugf("gpg: exporting gpg owner trust from host")

	ownerTrustExport, err := gpg.GetHostOwnerTrust()
	if err != nil {
		return fmt.Errorf("export local ownertrust from GPG: %w", err)
	}

	log.Debugf("gpg: detecting gpg-agent socket path on host")
	// Detect local agent extra socket, this will be forwarded to the remote and
	// symlinked in multiple paths
	gpgExtraSocketBytes, err := exec.Command("gpgconf", []string{"--list-dir", "agent-extra-socket"}...).
		Output()
	if err != nil {
		return err
	}

	gpgExtraSocketPath := strings.TrimSpace(string(gpgExtraSocketBytes))
	log.Debugf("gpg: detected gpg-agent socket path %s", gpgExtraSocketPath)

	gitGpgKey, err := exec.Command("git", []string{"config", "user.signingKey"}...).Output()
	if err != nil {
		log.Debugf("gpg: no git signkey detected, skipping")
	}
	log.Debugf("gpg: detected git sign key %s", gitGpgKey)

	log.Debugf("ssh: starting reverse forwarding socket %s", gpgExtraSocketPath)
	cmd.ReverseForwardPorts = append(cmd.ReverseForwardPorts, gpgExtraSocketPath)

	go func() {
		err := cmd.reverseForwardPorts(ctx, containerClient, log)
		if err != nil {
			log.Fatal(err)
		}
	}()

	pubKeyArgument := base64.StdEncoding.EncodeToString(pubKeyExport)
	ownerTrustArgument := base64.StdEncoding.EncodeToString(ownerTrustExport)

	// Now we forward the agent socket to the remote, and setup remote gpg to use it
	// fix eventual permissions and so on
	forwardAgent := []string{
		agent.ContainerDevPodHelperLocation,
	}

	if log.GetLevel() == logrus.DebugLevel {
		forwardAgent = append(forwardAgent, "--debug")
	}

	forwardAgent = append(forwardAgent, []string{
		"agent",
		"workspace",
		"setup-gpg",
		"--publickey",
		pubKeyArgument,
		"--ownertrust",
		ownerTrustArgument,
		"--socketpath",
		gpgExtraSocketPath,
	}...)

	if len(gitGpgKey) > 0 {
		forwardAgent = append(forwardAgent, "--gitkey")
		forwardAgent = append(forwardAgent, string(gitGpgKey))
	}

	log.Debugf(
		"gpg: start reverse forward of gpg-agent socket %s, keeping connection open",
		gpgExtraSocketPath,
	)

	command := strings.Join(forwardAgent, " ")

	if cmd.User != "" && cmd.User != "root" {
		command = fmt.Sprintf("su -c \"%s\" '%s'", command, cmd.User)
	}

	return devssh.Run(ctx, containerClient, command, nil, writer, writer)
}

func mergeDevPodSshOptions(cmd *SSHCmd) error {
	_, err := clientimplementation.DecodeOptionsFromEnv(
		clientimplementation.DevPodFlagsSsh,
		cmd,
	)
	if err != nil {
		return fmt.Errorf("decode up options: %w", err)
	}

	return nil
}

func startWorkspaceCredentialServer(ctx context.Context, client *ssh.Client, user string, allowGitCredentials, allowDockerCredentials bool, stdin io.Reader, stdout io.Writer, log log.Logger) error {
	writer := log.ErrorStreamOnly().Writer(logrus.DebugLevel, false)
	defer writer.Close()

	command := fmt.Sprintf("'%s' agent container credentials-server", agent.ContainerDevPodHelperLocation)
	args := []string{
		fmt.Sprintf("--user '%s'", user),
	}
	if allowGitCredentials {
		args = append(args, "--configure-git-helper")
	}
	if allowDockerCredentials {
		args = append(args, "--configure-docker-helper")
	}
	if log.GetLevel() == logrus.DebugLevel {
		args = append(args, "--debug")
	}
	args = append(args, "--runner")
	command = fmt.Sprintf("%s %s", command, strings.Join(args, " "))

	if err := devssh.Run(ctx, client, command, stdin, stdout, writer); err != nil {
		return fmt.Errorf("run credentials server: %w", err)
	}

	return nil
}

func startLocalServer(ctx context.Context, allowGitCredentials, allowDockerCredentials bool, gitUsername, gitToken string, stdoutReader io.Reader, stdinWriter io.WriteCloser, log log.Logger) error {
	if err := tunnelserver.RunRunnerServer(ctx, stdoutReader, stdinWriter, allowGitCredentials, allowDockerCredentials, gitUsername, gitToken, log); err != nil {
		return fmt.Errorf("run runner services server: %w", err)
	}

	return nil
}

func preparePipes() (io.Reader, io.WriteCloser, io.Reader, io.WriteCloser, error) {
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("create stdout pipe: %w", err)
	}
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("create stdin pipe: %w", err)
	}

	return stdoutReader, stdoutWriter, stdinReader, stdinWriter, nil
}
