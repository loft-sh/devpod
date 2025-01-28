package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/loft-sh/devpod/cmd/agent/workspace"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/cmd/machine"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/agent/tunnelserver"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	dpFlags "github.com/loft-sh/devpod/pkg/flags"
	"github.com/loft-sh/devpod/pkg/gpg"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/port"
	"github.com/loft-sh/devpod/pkg/provider"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	sshServer "github.com/loft-sh/devpod/pkg/ssh/server"
	"github.com/loft-sh/devpod/pkg/tailscale"
	"github.com/loft-sh/devpod/pkg/tunnel"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

const (
	DisableSSHKeepAlive time.Duration = 0 * time.Second
)

// SSHCmd holds the ssh cmd flags
type SSHCmd struct {
	*flags.GlobalFlags
	dpFlags.GitCredentialsFlags

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

	Proxy bool

	Command      string
	User         string
	WorkDir      string
	UseTailscale bool
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
			if err := mergeDevPodSshOptions(cmd); err != nil {
				return err
			}
			if cmd.Proxy {
				// merge context options from env
				config.MergeContextOptions(devPodConfig.Current(), os.Environ())
			}

			ctx := cobraCmd.Context()
			client, err := workspace2.Get(ctx, devPodConfig, args, true, log.Default.ErrorStreamOnly())
			if err != nil {
				return err
			}

			return cmd.Run(ctx, devPodConfig, client, log.Default.ErrorStreamOnly())
		},
	}

	dpFlags.SetGitCredentialsFlags(sshCmd.Flags(), &cmd.GitCredentialsFlags)
	sshCmd.Flags().StringArrayVarP(&cmd.ForwardPorts, "forward-ports", "L", []string{}, "Specifies that connections to the given TCP port or Unix socket on the local (client) host are to be forwarded to the given host and port, or Unix socket, on the remote side.")
	sshCmd.Flags().StringArrayVarP(&cmd.ReverseForwardPorts, "reverse-forward-ports", "R", []string{}, "Specifies that connections to the given TCP port or Unix socket on the local (client) host are to be reverse forwarded to the given host and port, or Unix socket, on the remote side.")
	sshCmd.Flags().StringArrayVarP(&cmd.SendEnvVars, "send-env", "", []string{}, "Specifies which local env variables shall be sent to the container.")
	sshCmd.Flags().StringArrayVarP(&cmd.SetEnvVars, "set-env", "", []string{}, "Specifies env variables to be set in the container.")
	sshCmd.Flags().StringVar(&cmd.ForwardPortsTimeout, "forward-ports-timeout", "", "Specifies the timeout after which the command should terminate when the ports are unused.")
	sshCmd.Flags().StringVar(&cmd.Command, "command", "", "The command to execute within the workspace")
	sshCmd.Flags().StringVar(&cmd.User, "user", "", "The user of the workspace to use")
	sshCmd.Flags().StringVar(&cmd.WorkDir, "workdir", "", "The working directory in the container")
	sshCmd.Flags().BoolVar(&cmd.Proxy, "proxy", false, "If true will act as intermediate proxy for a proxy provider")
	sshCmd.Flags().BoolVar(&cmd.AgentForwarding, "agent-forwarding", true, "If true forward the local ssh keys to the remote machine")
	sshCmd.Flags().StringVar(&cmd.ReuseSSHAuthSock, "reuse-ssh-auth-sock", "", "If set, the SSH_AUTH_SOCK is expected to already be available in the workspace (under /tmp using the key provided) and the connection reuses this instead of creating a new one")
	_ = sshCmd.Flags().MarkHidden("reuse-ssh-auth-sock")
	sshCmd.Flags().BoolVar(&cmd.GPGAgentForwarding, "gpg-agent-forwarding", false, "If true forward the local gpg-agent to the remote machine")
	sshCmd.Flags().BoolVar(&cmd.Stdio, "stdio", false, "If true will tunnel connection through stdout and stdin")
	sshCmd.Flags().BoolVar(&cmd.StartServices, "start-services", true, "If false will not start any port-forwarding or git / docker credentials helper")
	sshCmd.Flags().DurationVar(&cmd.SSHKeepAliveInterval, "ssh-keepalive-interval", 55*time.Second, "How often should keepalive request be made (55s)")
	sshCmd.Flags().BoolVar(&cmd.UseTailscale, "use-tailscale", true, "")

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

	if cmd.UseTailscale {
		proxyClient, ok := client.(client2.ProxyClient)
		if !ok {
			return fmt.Errorf("tailscale is only accessible to DevPod Pro workspaces")
		}
		return cmd.startTSProxyTunnel(ctx, devPodConfig, proxyClient, log)
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
			return cmd.startTunnel(ctx, devPodConfig, containerClient, client, log)
		},
	)
}

func (cmd *SSHCmd) startTSProxyTunnel(
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.ProxyClient,
	log log.Logger,
) error {
	log.Debugf("Starting proxy connection")

	if cmd.Provider == "" {
		cmd.Provider = devPodConfig.Current().DefaultProvider
	}

	// Load configuration
	config, err := platform.ReadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return fmt.Errorf("failed to read devpod configuration: %w", err)
	}

	// Get client hostname
	hostname, err := tailscale.GetHostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %w", err)
	}

	// Initialize Tailscale network
	network := tailscale.NewTSNet(&tailscale.TSNetConfig{
		AccessKey: config.AccessKey,
		Host:      tailscale.RemoveProtocol(config.Host),
		Hostname:  hostname,
		LogF:      func(format string, args ...any) {},
	})

	// Start Tailscale network
	startCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		if err := network.Start(startCtx); err != nil {
			log.Errorf("failed to start TSNet: %v", err)
			cancel()
		}
	}()

	// Wait for the host to become reachable
	reachableHost := fmt.Sprintf(
		"devpod.%v.%v.test.workspace",
		client.WorkspaceConfig().Pro.DisplayName,
		client.WorkspaceConfig().Pro.Project,
	)
	if err := waitUntilReachable(ctx, network, reachableHost, log); err != nil {
		return fmt.Errorf("failed to reach TSNet host: %w", err)
	}

	log.Debugf("Host %s is reachable. Proceeding with SSH session...", reachableHost)

	return runTSTunnel(
		ctx,
		network,
		reachableHost,
		cmd,
		devPodConfig,
		client,
		log,
	)
}

// TODO: move this to pkg/tunnel
func runTSTunnel(
	ctx context.Context,
	network tailscale.TSNet,
	host string,
	cmd *SSHCmd, // FIXME: ???
	devPodConfig *config.Config,
	workspaceClient client2.ProxyClient,
	log log.Logger,
) error {
	return establishOuterTunnel(ctx, network, host, log, func(outerSshClient *ssh.Client) error {
		log.Debugf("Outer tunnel established. Starting inner server...")

		// Start the inner SSH server
		if err := startInnerServer(outerSshClient, cmd.User, cmd.WorkDir, log); err != nil {
			log.Errorf("Failed to start inner SSH server: %v", err)
			return fmt.Errorf("failed to start inner SSH server: %w", err)
		}

		// Wait for the inner server to be reachable
		// FIXME: Ensure this works correctly, considering the reverse proxy server on this port.
		if err := waitForInnerServer(ctx, network, host, sshServer.DefaultUserPort); err != nil {
			log.Errorf("Inner server not reachable: %v", err)
			return fmt.Errorf("inner SSH server not reachable: %w", err)
		}

		log.Debugf("Inner server reachable. Establishing user session...")

		// Create an SSH client for the inner server
		sshClient, err := devssh.TailscaleClientWithUser(
			ctx, network, tailscale.GetURL(host, sshServer.DefaultUserPort), cmd.User, log)
		if err != nil {
			return fmt.Errorf("failed to create SSH client for inner server: %w", err)
		}
		defer sshClient.Close()

		// Forward ports if specified
		if len(cmd.ForwardPorts) > 0 {
			return cmd.forwardPorts(ctx, outerSshClient, log)
		}

		// Reverse forward ports if specified
		if len(cmd.ReverseForwardPorts) > 0 && !cmd.GPGAgentForwarding {
			return cmd.reverseForwardPorts(ctx, outerSshClient, log)
		}

		// Start port-forwarding and services if enabled
		if cmd.StartServices {
			go cmd.startServices(ctx, devPodConfig, outerSshClient, cmd.GitUsername, cmd.GitToken, workspaceClient.WorkspaceConfig(), log)
		}

		// Handle GPG agent forwarding
		if cmd.GPGAgentForwarding || devPodConfig.ContextOption(config.ContextOptionGPGAgentForwarding) == "true" {
			if gpg.IsGpgTunnelRunning(cmd.User, ctx, sshClient, log) {
				log.Debugf("[GPG] exporting already running, skipping")
			} else if err := cmd.setupGPGAgent(ctx, sshClient, log); err != nil {
				return err
			}
		}

		// Retrieve environment variables
		// envVars, err := cmd.retrieveEnVars()
		// if err != nil {
		// 	return err
		// }

		// Handle ssh remote proxy mode
		if cmd.Stdio {
			if cmd.SSHKeepAliveInterval != DisableSSHKeepAlive {
				go startSSHKeepAlive(ctx, outerSshClient, cmd.SSHKeepAliveInterval, log)
			}

			go func() {
				if err := cmd.startRunnerServices(ctx, devPodConfig, outerSshClient, log); err != nil {
					log.Error(err)
				}
			}()

			go func() {
				cmd.setupPlatformAccess(ctx, sshClient, log) // FIXME: for whatever reason this doesn't work but when running command from inside the container it works just fine
			}()

			return tailscale.DirectTunnel(ctx, network, host, sshServer.DefaultUserPort, os.Stdin, os.Stdout)
		}

		// Connect to the inner server and handle user session
		return machine.RunSSHSession(
			ctx,
			sshClient,
			cmd.AgentForwarding,
			cmd.Command,
			os.Stderr,
		)
	})
}

// establishOuterTunnel establishes the root-level connection.
func establishOuterTunnel(ctx context.Context, network tailscale.TSNet, host string, log log.Logger, onReady func(*ssh.Client) error) error {
	log.Debugf("Estabilishing outer connection")
	address := tailscale.GetURL(host, sshServer.DefaultPort)
	conn, err := network.Dial(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to outer tunnel: %w", err)
	}
	defer conn.Close()

	sshConfig := &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	clientConn, chans, reqs, err := ssh.NewClientConn(conn, address, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to establish SSH connection: %w", err)
	}
	client := ssh.NewClient(clientConn, chans, reqs)
	defer client.Close()

	log.Debugf("Outer connection ready")
	return onReady(client)
}

// startInnerServer starts the user-level SSH server.
func startInnerServer(sshClient *ssh.Client, user, workDir string, log log.Logger) error {
	command := fmt.Sprintf(
		"/usr/local/bin/devpod helper ssh-server --workdir '%s' --address 0.0.0.0:%d",
		workDir, sshServer.DefaultUserPort,
	)
	// if user != "root" {
	// 	command = fmt.Sprintf("su -c \"%s\" '%s'", command, user)
	// }

	session, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	var stderr bytes.Buffer
	session.Stderr = &stderr

	if err := session.Run(command); err != nil {
		return fmt.Errorf("inner server start failed: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// waitForInnerServer waits for the inner server to become reachable.
func waitForInnerServer(ctx context.Context, network tailscale.TSNet, host string, port int) error {
	address := tailscale.GetURL(host, port)
	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		conn, err := network.Dial(ctx, "tcp", address)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("inner server not reachable at %s", address)
}

// waitUntilReachable polls until the given host is reachable via Tailscale.
func waitUntilReachable(ctx context.Context, ts tailscale.TSNet, host string, log log.Logger) error {
	const retryInterval = 100 * time.Millisecond
	const maxRetries = 60

	for i := 0; i < maxRetries; i++ {
		conn, err := ts.Dial(ctx, "tcp", tailscale.GetURL(host, sshServer.DefaultPort))
		if err == nil {
			_ = conn.Close()
			return nil // Host is reachable
		}
		log.Debugf("Host %s not reachable, retrying... (%d/%d)", host, i+1, maxRetries)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryInterval):
		}
	}

	return fmt.Errorf("host %s not reachable after %d attempts", host, maxRetries)
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

	// We can optimize if we know we're on pro and the client is local
	if cmd.Proxy && client.AgentLocal() {
		encodedWorkspaceInfo, _, err := client.AgentInfo(provider.CLIOptions{Proxy: true})
		if err != nil {
			return fmt.Errorf("prepare workspace info: %w", err)
		}
		// we don't need the client anymore, can unlock
		client.Unlock()

		shouldExit, workspaceInfo, err := agent.WorkspaceInfo(encodedWorkspaceInfo, log)
		if err != nil {
			return err
		} else if shouldExit {
			return nil
		}

		return cmd.jumpLocalProxyContainer(ctx, devPodConfig, workspaceInfo, log, func(ctx context.Context, command string, sshClient *ssh.Client) error {
			writer := log.Writer(logrus.InfoLevel, false)
			defer writer.Close()

			return devssh.Run(ctx, sshClient, command, os.Stdin, os.Stdout, writer, nil)
		})
	}

	// tunnel to container
	return tunnel.NewContainerTunnel(client, cmd.Proxy, log).
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

	// start port-forwarding etc.
	if !cmd.Proxy && cmd.StartServices {
		go cmd.startServices(ctx, devPodConfig, containerClient, cmd.GitUsername, cmd.GitToken, workspaceClient.WorkspaceConfig(), log)
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
	command := fmt.Sprintf("'%s' helper ssh-server --track-activity --workdir '%s'", agent.ContainerDevPodHelperLocation, workdir)
	if cmd.ReuseSSHAuthSock != "" {
		log.Debug("Reusing SSH_AUTH_SOCK")
		command += fmt.Sprintf(" --reuse-ssh-auth-sock=%s", cmd.ReuseSSHAuthSock)
	}
	if cmd.Debug {
		command += " --debug"
	}
	if !cmd.Proxy && cmd.User != "" && cmd.User != "root" {
		command = fmt.Sprintf("su -c \"%s\" '%s'", command, cmd.User)
	}

	envVars, err := cmd.retrieveEnVars()
	if err != nil {
		return err
	}

	// Traffic is coming in from the outside, we need to forward it to the container
	if cmd.Proxy || cmd.Stdio {
		if cmd.Proxy {
			if cmd.SSHKeepAliveInterval != DisableSSHKeepAlive {
				go startSSHKeepAlive(ctx, containerClient, cmd.SSHKeepAliveInterval, log)
			}

			go func() {
				if err := cmd.startRunnerServices(ctx, devPodConfig, containerClient, log); err != nil {
					log.Error(err)
				}
			}()

			go func() {
				cmd.setupPlatformAccess(ctx, containerClient, log)
			}()
		}
		return devssh.Run(ctx, containerClient, command, os.Stdin, os.Stdout, writer, envVars)
	}

	return machine.StartSSHSession(
		ctx,
		cmd.User,
		cmd.Command,
		!cmd.Proxy && cmd.AgentForwarding &&
			devPodConfig.ContextOption(config.ContextOptionSSHAgentForwarding) == "true",
		func(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			if cmd.SSHKeepAliveInterval != DisableSSHKeepAlive {
				go startSSHKeepAlive(ctx, containerClient, cmd.SSHKeepAliveInterval, log)
			}
			return devssh.Run(ctx, containerClient, command, stdin, stdout, stderr, envVars)
		},
		writer,
	)
}

func (cmd *SSHCmd) setupPlatformAccess(ctx context.Context, sshClient *ssh.Client, log log.Logger) {
	buf := &bytes.Buffer{}
	command := fmt.Sprintf("'%s' agent container setup-loft-platform-access", agent.ContainerDevPodHelperLocation)
	err := devssh.Run(ctx, sshClient, command, nil, buf, buf, nil)
	if err != nil {
		log.Debugf("Failed to setup platform access: %s%v", buf.String(), err)
	}
}

func (cmd *SSHCmd) startServices(
	ctx context.Context,
	devPodConfig *config.Config,
	containerClient *ssh.Client,
	gitUsername,
	gitToken string,
	workspace *provider.Workspace,
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
			gitUsername,
			gitToken,
			workspace,
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
	return retry.OnError(wait.Backoff{
		Steps:    math.MaxInt,
		Duration: 200 * time.Millisecond,
		Factor:   1,
		Jitter:   0.1,
	}, func(err error) bool {
		if ctx.Err() != nil {
			log.Infof("Context canceled, stopping credentials server: %v", ctx.Err())
			return false
		}
		return true
	}, func() error {
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
	})
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

// jumpLocalProxyContainer is a shortcut we can take if we have a local provider and we're in proxy mode.
// This completely skips the agent.
//
// WARN: This is considered experimental for the time being!
func (cmd *SSHCmd) jumpLocalProxyContainer(ctx context.Context, devPodConfig *config.Config, workspaceInfo *provider.AgentWorkspaceInfo, log log.Logger, exec func(ctx context.Context, command string, sshClient *ssh.Client) error) error {
	_, err := workspace.InitContentFolder(workspaceInfo, log)
	if err != nil {
		return err
	}

	runner, err := workspace.CreateRunner(workspaceInfo, nil, log)
	if err != nil {
		return err
	}

	containerDetails, err := runner.Find(ctx)
	if err != nil {
		return err
	}

	if containerDetails == nil || containerDetails.State.Status != "running" {
		log.Info("Workspace isn't running, starting up...")
		_, err := runner.Up(ctx, devcontainer.UpOptions{NoBuild: true}, workspaceInfo.InjectTimeout)
		if err != nil {
			return err
		}
		log.Info("Successfully started workspace")
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
	defer stdoutWriter.Close()
	defer stdinWriter.Close()

	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		writer := log.Writer(logrus.InfoLevel, false)
		command := fmt.Sprintf("'%s' helper ssh-server --stdio", agent.ContainerDevPodHelperLocation)
		if log.GetLevel() == logrus.DebugLevel {
			command += " --debug"
		}

		err := runner.Command(cancelCtx, "root", command, stdinReader, stdoutWriter, writer)
		if err != nil {
			errChan <- err
		}
	}()

	containerClient, err := devssh.StdioClient(stdoutReader, stdinWriter, false)
	if err != nil {
		return err
	}
	defer containerClient.Close()

	if len(cmd.ForwardPorts) > 0 {
		return cmd.forwardPorts(ctx, containerClient, log)
	}

	if len(cmd.ReverseForwardPorts) > 0 && !cmd.GPGAgentForwarding {
		return cmd.reverseForwardPorts(ctx, containerClient, log)
	}

	go startSSHKeepAlive(ctx, containerClient, cmd.SSHKeepAliveInterval, log)
	go cmd.setupPlatformAccess(ctx, containerClient, log)
	go func() {
		if err := cmd.startRunnerServices(ctx, devPodConfig, containerClient, log); err != nil {
			log.Error(err)
		}
	}()

	workdir := filepath.Join("/workspaces", workspaceInfo.Workspace.ID)
	if cmd.WorkDir != "" {
		workdir = cmd.WorkDir
	}
	command := fmt.Sprintf("'%s' helper ssh-server --track-activity --stdio --workdir '%s'", agent.ContainerDevPodHelperLocation, workdir)
	if cmd.Debug {
		command += " --debug"
	}
	go func() {
		errChan <- exec(cancelCtx, command, containerClient)
	}()

	return <-errChan
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

	return devssh.Run(ctx, client, command, stdin, stdout, writer, nil)
}

func startLocalServer(ctx context.Context, allowGitCredentials, allowDockerCredentials bool, gitUsername, gitToken string, stdoutReader io.Reader, stdinWriter io.WriteCloser, log log.Logger) error {
	err := tunnelserver.RunRunnerServer(ctx, stdoutReader, stdinWriter, allowGitCredentials, allowDockerCredentials, gitUsername, gitToken, log)
	if err != nil {
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
