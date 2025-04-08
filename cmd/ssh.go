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
	"time"

	"github.com/loft-sh/devpod/cmd/completion"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/cmd/machine"
	"github.com/loft-sh/devpod/pkg/agent"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	daemon "github.com/loft-sh/devpod/pkg/daemon/platform"
	"github.com/loft-sh/devpod/pkg/gpg"
	"github.com/loft-sh/devpod/pkg/port"
	"github.com/loft-sh/devpod/pkg/provider"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/tunnel"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

const (
	DisableSSHKeepAlive time.Duration = 0 * time.Second
)

// SSHCmd holds the ssh cmd flags
type SSHCmd struct {
	*flags.GlobalFlags

	ForwardPortsTimeout string
	ForwardPorts        []string
	ReverseForwardPorts []string
	SendEnvVars         []string
	SetEnvVars          []string

	Stdio                     bool
	JumpContainer             bool
	ReuseSSHAuthSock          string
	AgentForwarding           bool
	GPGAgentForwarding        bool
	GitSSHSignatureForwarding bool

	// ssh keepalive options
	SSHKeepAliveInterval time.Duration `json:"sshKeepAliveInterval,omitempty"`

	StartServices bool

	Command string
	User    string
	WorkDir string
}

// NewSSHCmd creates a new ssh command
func NewSSHCmd(f *flags.GlobalFlags) *cobra.Command {
	cmd := &SSHCmd{
		GlobalFlags: f,
	}
	sshCmd := &cobra.Command{
		Use:   "ssh [flags] [workspace-folder|workspace-name]",
		Short: "Starts a new ssh session to a workspace",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
			if err != nil {
				return err
			}

			localOnly := false
			if cmd.Stdio {
				localOnly = true
			}

			ctx := cobraCmd.Context()
			client, err := workspace2.Get(ctx, devPodConfig, args, true, cmd.Owner, localOnly, log.Default.ErrorStreamOnly())
			if err != nil {
				return err
			}

			return cmd.Run(ctx, devPodConfig, client, log.Default.ErrorStreamOnly())
		},
		ValidArgsFunction: func(rootCmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completion.GetWorkspaceSuggestions(rootCmd, cmd.Context, cmd.Provider, args, toComplete, cmd.Owner, log.Default)
		},
	}

	sshCmd.Flags().StringArrayVarP(&cmd.ForwardPorts, "forward-ports", "L", []string{}, "Specifies that connections to the given TCP port or Unix socket on the local (client) host are to be forwarded to the given host and port, or Unix socket, on the remote side.")
	sshCmd.Flags().StringArrayVarP(&cmd.ReverseForwardPorts, "reverse-forward-ports", "R", []string{}, "Specifies that connections to the given TCP port or Unix socket on the local (client) host are to be reverse forwarded to the given host and port, or Unix socket, on the remote side.")
	sshCmd.Flags().StringArrayVarP(&cmd.SendEnvVars, "send-env", "", []string{}, "Specifies which local env variables shall be sent to the container.")
	sshCmd.Flags().StringArrayVarP(&cmd.SetEnvVars, "set-env", "", []string{}, "Specifies env variables to be set in the container.")
	sshCmd.Flags().StringVar(&cmd.ForwardPortsTimeout, "forward-ports-timeout", "", "Specifies the timeout after which the command should terminate when the ports are unused.")
	sshCmd.Flags().StringVar(&cmd.Command, "command", "", "The command to execute within the workspace")
	sshCmd.Flags().StringVar(&cmd.User, "user", "", "The user of the workspace to use")
	sshCmd.Flags().StringVar(&cmd.WorkDir, "workdir", "", "The working directory in the container")
	sshCmd.Flags().BoolVar(&cmd.AgentForwarding, "agent-forwarding", true, "If true forward the local ssh keys to the remote machine")
	sshCmd.Flags().StringVar(&cmd.ReuseSSHAuthSock, "reuse-ssh-auth-sock", "", "If set, the SSH_AUTH_SOCK is expected to already be available in the workspace (under /tmp using the key provided) and the connection reuses this instead of creating a new one")
	_ = sshCmd.Flags().MarkHidden("reuse-ssh-auth-sock")
	sshCmd.Flags().BoolVar(&cmd.GPGAgentForwarding, "gpg-agent-forwarding", false, "If true forward the local gpg-agent to the remote machine")
	sshCmd.Flags().BoolVar(&cmd.Stdio, "stdio", false, "If true will tunnel connection through stdout and stdin")
	sshCmd.Flags().BoolVar(&cmd.StartServices, "start-services", true, "If false will not start any port-forwarding or git / docker credentials helper")
	sshCmd.Flags().DurationVar(&cmd.SSHKeepAliveInterval, "ssh-keepalive-interval", 55*time.Second, "How often should keepalive request be made (55s)")

	return sshCmd
}

// Run runs the command logic
func (cmd *SSHCmd) Run(
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.BaseWorkspaceClient,
	log log.Logger) error {
	// add ssh keys to agent
	if devPodConfig.ContextOption(config.ContextOptionSSHAgentForwarding) == "true" && devPodConfig.ContextOption(config.ContextOptionSSHAddPrivateKeys) == "true" {
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

	workspaceClient, ok := client.(client2.WorkspaceClient)
	if ok {
		return cmd.jumpContainer(ctx, devPodConfig, workspaceClient, log)
	}
	proxyClient, ok := client.(client2.ProxyClient)
	if ok {
		return cmd.startProxyTunnel(ctx, devPodConfig, proxyClient, log)
	}
	daemonClient, ok := client.(client2.DaemonClient)
	if ok {
		return cmd.jumpContainerTailscale(ctx, devPodConfig, daemonClient, log)
	}

	return nil
}

func (cmd *SSHCmd) jumpContainerTailscale(
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.DaemonClient,
	log log.Logger,
) error {
	log.Debugf("Starting tailscale connection")

	err := client.CheckWorkspaceReachable(ctx)
	if err != nil {
		return err
	}

	toolSSHClient, sshClient, err := client.SSHClients(ctx, cmd.User)
	if err != nil {
		return err
	}
	defer toolSSHClient.Close()
	defer sshClient.Close()

	// Forward ports if specified
	if len(cmd.ForwardPorts) > 0 {
		return cmd.forwardPorts(ctx, toolSSHClient, log)
	}

	// Reverse forward ports if specified
	if len(cmd.ReverseForwardPorts) > 0 && !cmd.GPGAgentForwarding {
		return cmd.reverseForwardPorts(ctx, toolSSHClient, log)
	}

	if cmd.StartServices {
		go func() {
			err = startServicesDaemon(ctx,
				devPodConfig,
				client,
				toolSSHClient,
				cmd.User,
				log,
				false,
				nil,
			)
			if err != nil {
				log.Errorf("Error starting services: %v", err)
			}
		}()
	}

	// Handle GPG agent forwarding
	if cmd.GPGAgentForwarding || devPodConfig.ContextOption(config.ContextOptionGPGAgentForwarding) == "true" {
		if gpg.IsGpgTunnelRunning(cmd.User, ctx, toolSSHClient, log) {
			log.Debugf("[GPG] exporting already running, skipping")
		} else if err := cmd.setupGPGAgent(ctx, toolSSHClient, log); err != nil {
			return err
		}
	}

	// Handle ssh stdio mode
	if cmd.Stdio {
		if cmd.SSHKeepAliveInterval != DisableSSHKeepAlive {
			go startSSHKeepAlive(ctx, toolSSHClient, cmd.SSHKeepAliveInterval, log)
		}

		return client.DirectTunnel(ctx, os.Stdin, os.Stdout)
	}

	// Connect to the inner server and handle user session
	return machine.RunSSHSession(
		ctx,
		sshClient,
		cmd.AgentForwarding,
		cmd.Command,
		os.Stderr,
	)
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
			return cmd.startTunnel(ctx, devPodConfig, containerClient, client, log)
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

func (cmd *SSHCmd) retrieveEnVars() (map[string]string, error) {
	envVars := make(map[string]string)
	for _, envVar := range cmd.SendEnvVars {
		envVarValue, exist := os.LookupEnv(envVar)
		if exist {
			envVars[envVar] = envVarValue
		}
	}
	for _, envVar := range cmd.SetEnvVars {
		parts := strings.Split(envVar, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid env var: %s", envVar)
		}
		envVars[parts[0]] = parts[1]
	}

	return envVars, nil
}

func (cmd *SSHCmd) jumpContainer(
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.WorkspaceClient,
	log log.Logger,
) error {
	// lock the workspace as long as we init the connection
	err := client.Lock(ctx)
	if err != nil {
		return err
	}
	defer client.Unlock()

	// start the workspace
	err = startWait(ctx, client, false, log)
	if err != nil {
		return err
	}

	envVars, err := cmd.retrieveEnVars()
	if err != nil {
		return err
	}

	// tunnel to container
	return tunnel.NewContainerTunnel(client, log).
		Run(ctx, func(ctx context.Context, containerClient *ssh.Client) error {
			// we have a connection to the container, make sure others can connect as well
			client.Unlock()

			// start ssh tunnel
			return cmd.startTunnel(ctx, devPodConfig, containerClient, client, log)
		}, devPodConfig, envVars)
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
			if !errors.Is(io.EOF, err) {
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
			if !errors.Is(io.EOF, err) {
				errChan <- fmt.Errorf("error forwarding %s: %w", portMapping, err)
			}
		}(portMapping)
	}

	return <-errChan
}

func (cmd *SSHCmd) startTunnel(ctx context.Context, devPodConfig *config.Config, containerClient *ssh.Client, workspaceClient client2.BaseWorkspaceClient, log log.Logger) error {
	// check if we should forward ports
	if len(cmd.ForwardPorts) > 0 {
		return cmd.forwardPorts(ctx, containerClient, log)
	}

	// check if we should reverse forward ports
	if len(cmd.ReverseForwardPorts) > 0 && !cmd.GPGAgentForwarding {
		return cmd.reverseForwardPorts(ctx, containerClient, log)
	}

	if cmd.StartServices {
		configureDockerCredentials := devPodConfig.ContextOption(config.ContextOptionSSHInjectDockerCredentials) == "true"
		configureGitCredentials := devPodConfig.ContextOption(config.ContextOptionSSHInjectGitCredentials) == "true"
		configureGitSSHSignatureHelper := devPodConfig.ContextOption(config.ContextOptionGitSSHSignatureForwarding) == "true"

		go cmd.startServices(ctx, devPodConfig, containerClient, workspaceClient.WorkspaceConfig(), configureDockerCredentials, configureGitCredentials, configureGitSSHSignatureHelper, log)
	}
	// start ssh
	writer := log.ErrorStreamOnly().Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// check if we should do gpg agent forwarding
	if cmd.GPGAgentForwarding || devPodConfig.ContextOption(config.ContextOptionGPGAgentForwarding) == "true" {
		// Check if a forwarding is already enabled and running, in that case
		// we skip the forwarding and keep using the original one
		if gpg.IsGpgTunnelRunning(cmd.User, ctx, containerClient, log) {
			log.Debugf("[GPG] exporting already running, skipping")
		} else {
			err := cmd.setupGPGAgent(ctx, containerClient, log)
			if err != nil {
				return err
			}
		}
	}

	workdir := filepath.Join("/workspaces", workspaceClient.Workspace())
	if cmd.WorkDir != "" {
		workdir = cmd.WorkDir
	}

	log.Debugf("Run outer container tunnel")
	command := fmt.Sprintf("'%s' helper ssh-server --track-activity --stdio --workdir '%s'", agent.ContainerDevPodHelperLocation, workdir)
	if cmd.ReuseSSHAuthSock != "" {
		log.Debug("Reusing SSH_AUTH_SOCK")
		command += fmt.Sprintf(" --reuse-ssh-auth-sock=%s", cmd.ReuseSSHAuthSock)
	}
	if cmd.Debug {
		command += " --debug"
	}
	if cmd.User != "" && cmd.User != "root" {
		command = fmt.Sprintf("su -c \"%s\" '%s'", command, cmd.User)
	}

	envVars, err := cmd.retrieveEnVars()
	if err != nil {
		return err
	}

	// Traffic is coming in from the outside, we need to forward it to the container
	if cmd.Stdio {
		return devssh.Run(ctx, containerClient, command, os.Stdin, os.Stdout, writer, envVars)
	}

	return machine.StartSSHSession(
		ctx,
		cmd.User,
		cmd.Command,
		cmd.AgentForwarding && devPodConfig.ContextOption(config.ContextOptionSSHAgentForwarding) == "true",
		func(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			if cmd.SSHKeepAliveInterval != DisableSSHKeepAlive {
				go startSSHKeepAlive(ctx, containerClient, cmd.SSHKeepAliveInterval, log)
			}
			return devssh.Run(ctx, containerClient, command, stdin, stdout, stderr, envVars)
		},
		writer,
	)
}

func (cmd *SSHCmd) startServices(
	ctx context.Context,
	devPodConfig *config.Config,
	containerClient *ssh.Client,
	workspace *provider.Workspace,
	configureDockerCredentials, configureGitCredentials, configureGitSSHSignatureHelper bool,
	log log.Logger,
) {
	if cmd.User != "" {
		err := tunnel.RunServices(
			ctx,
			devPodConfig,
			containerClient,
			cmd.User,
			false,
			nil,
			nil,
			workspace,
			configureDockerCredentials,
			configureGitCredentials,
			configureGitSSHSignatureHelper,
			log,
		)
		if err != nil {
			log.Debugf("Error running credential server: %v", err)
		}
	}
}

// setupGPGAgent will forward a local gpg-agent into the remote container
// this works by using cmd/agent/workspace/setup_gpg
func (cmd *SSHCmd) setupGPGAgent(
	ctx context.Context,
	containerClient *ssh.Client,
	log log.Logger,
) error {
	log.Debugf("[GPG] exporting gpg owner trust from host")
	ownerTrustExport, err := gpg.GetHostOwnerTrust()
	if err != nil {
		return fmt.Errorf("export local ownertrust from GPG: %w", err)
	}
	ownerTrustArgument := base64.StdEncoding.EncodeToString(ownerTrustExport)

	log.Debugf("[GPG] detecting gpg-agent socket path on host")
	// Detect local agent extra socket, this will be forwarded to the remote and
	// symlinked in multiple paths
	gpgExtraSocketBytes, err := exec.Command("gpgconf", []string{"--list-dir", "agent-extra-socket"}...).
		Output()
	if err != nil {
		return err
	}

	gpgExtraSocketPath := strings.TrimSpace(string(gpgExtraSocketBytes))
	log.Debugf("[GPG] detected gpg-agent socket path %s", gpgExtraSocketPath)

	gitGpgKey, err := exec.Command("git", []string{"config", "user.signingKey"}...).Output()
	if err != nil {
		log.Debugf("[GPG] no git signkey detected, skipping")
	} else {
		log.Debugf("[GPG] detected git sign key %s", gitGpgKey)
	}

	cmd.ReverseForwardPorts = append(cmd.ReverseForwardPorts, gpgExtraSocketPath)

	// Now we forward the agent socket to the remote, and setup remote gpg to use it
	forwardAgent := []string{
		agent.ContainerDevPodHelperLocation,
		"agent",
		"workspace",
		"setup-gpg",
		"--ownertrust",
		ownerTrustArgument,
		"--socketpath",
		gpgExtraSocketPath,
	}

	if log.GetLevel() == logrus.DebugLevel {
		forwardAgent = append(forwardAgent, "--debug")
	}

	if len(gitGpgKey) > 0 {
		gitKey := strings.TrimSpace(string(gitGpgKey))
		forwardAgent = append(forwardAgent, "--gitkey")
		forwardAgent = append(forwardAgent, fmt.Sprintf("'%s'", gitKey))
	}

	command := strings.Join(forwardAgent, " ")
	if cmd.User != "" && cmd.User != "root" {
		command = fmt.Sprintf("su -c \"%s\" '%s'", command, cmd.User)
	}

	log.Debugf(
		"[GPG] start reverse forward of gpg-agent socket %s, keeping connection open",
		gpgExtraSocketPath,
	)

	go func() {
		log.Error(cmd.reverseForwardPorts(ctx, containerClient, log))
	}()

	writer := log.ErrorStreamOnly().Writer(logrus.InfoLevel, false)
	defer writer.Close()
	err = devssh.Run(ctx, containerClient, command, nil, writer, writer, nil)
	if err != nil {
		return fmt.Errorf("run gpg agent setup command: %w", err)
	}

	return nil
}

func startSSHKeepAlive(ctx context.Context, client *ssh.Client, interval time.Duration, log log.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				log.Errorf("Failed to send keepalive: %w", err)
			}
		}
	}
}

func startServicesDaemon(ctx context.Context, devPodConfig *config.Config, client client2.DaemonClient, sshClient *ssh.Client, user string, log log.Logger, forwardPorts bool, extraPorts []string) error {
	workspace, err := daemon.NewLocalClient(client.Provider()).GetWorkspace(ctx, client.WorkspaceConfig().UID)
	if err != nil {
		return err
	}

	configureDockerCredentials := devPodConfig.ContextOption(config.ContextOptionSSHInjectDockerCredentials) == "true"
	configureGitCredentials := devPodConfig.ContextOption(config.ContextOptionSSHInjectGitCredentials) == "true"
	configureGitSSHSignatureHelper := devPodConfig.ContextOption(config.ContextOptionGitSSHSignatureForwarding) == "true"

	if workspace != nil && workspace.Status.Instance != nil && workspace.Status.Instance.CredentialForwarding != nil {
		if workspace.Status.Instance.CredentialForwarding.Docker != nil {
			configureDockerCredentials = !workspace.Status.Instance.CredentialForwarding.Docker.Disabled
		}
		if workspace.Status.Instance.CredentialForwarding.Git != nil {
			configureGitCredentials = !workspace.Status.Instance.CredentialForwarding.Git.Disabled
			configureGitSSHSignatureHelper = !workspace.Status.Instance.CredentialForwarding.Git.Disabled
		}
	}

	if user != "" {
		return tunnel.RunServices(
			ctx,
			devPodConfig,
			sshClient,
			user,
			forwardPorts,
			extraPorts,
			nil,
			client.WorkspaceConfig(),
			configureDockerCredentials,
			configureGitCredentials,
			configureGitSSHSignatureHelper,
			log,
		)
	}

	return nil
}
