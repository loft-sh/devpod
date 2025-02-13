package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"syscall"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/devpod/cmd/machine"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/gpg"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/project"
	sshServer "github.com/loft-sh/devpod/pkg/ssh/server"
	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/takama/daemon"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tsClient "tailscale.com/client/tailscale"
)

func startTSProxyTunnel(
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.ProxyClient,
	cmd SSHCmd,
	log log.Logger,
) error {
	log.Debugf("Starting proxy connection")

	baseClient, err := platform.InitClientFromProvider(ctx, devPodConfig, client.WorkspaceConfig().Provider.Name, log)
	if err != nil {
		return err
	}

	daemonSocket, err := ts.GetSocketForProvider(devPodConfig, client.WorkspaceConfig().Provider.Name)
	if err != nil {
		return err
	}

	// TODO: Can we move this to the provider side?
	lc := &tsClient.LocalClient{
		Socket:        daemonSocket,
		UseSocketOnly: true,
	}

	log.Info("Check local node status")
	status, err := lc.Status(ctx)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ECONNREFUSED) {
			err = startDevPodD(ts.RemoveProtocol(baseClient.Config().Host))
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("failed to connect to local DevPod Daemon: %w", err)
		}
	}

	// TODO: handle not-authenticated state
	err = ts.WaitNodeReady(ctx, lc)
	if err != nil {
		return fmt.Errorf("wait node ready: %w", err)
	}

	err = ts.CheckLocalNodeReady(status)
	if err != nil {
		return fmt.Errorf("check local node ready: %w", err)
	}
	log.Done("Local node is ready")

	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	wCfg := client.WorkspaceConfig()
	ns := project.ProjectNamespace(wCfg.Pro.Project)
	// notify platform that we'd like to connect
	log.Info("Sending SSH request to platform")
	clientHostname, err := ts.GetOSHostname()
	if err != nil {
		log.Error("Error getting client hostname")
	}
	sshOpts := &managementv1.DevPodSshOptions{
		ClientHostname: clientHostname,
	}
	_, err = managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(ns).SSH(ctx, wCfg.Pro.InstanceName, sshOpts, metav1.CreateOptions{})
	if err != nil {
		log.Error("Error sending SSH Request", err)
	}

	wAddr := ts.NewAddr(ts.GetWorkspaceHostname(wCfg.Pro.InstanceName, wCfg.Pro.Project), sshServer.DefaultUserPort)
	log.Info("workspace host", wAddr.Host())

	err = ts.WaitHostReachable(ctx, lc, wAddr, log)
	if err != nil {
		return fmt.Errorf("failed to reach TSNet host: %w", err)
	}

	log.Debugf("Host %s is reachable. Proceeding with SSH session...", wAddr.Host())

	log.Info("Creating tool ssh client")
	// timeoutCtx, err :=
	// Create an SSH Client for the tool server
	toolSSHClient, err := ts.WaitForSSHClient(ctx, lc, wAddr.Host(), wAddr.Port(), "root", log)
	if err != nil {
		return fmt.Errorf("failed to create SSH client for tool server: %w", err)
	}
	defer toolSSHClient.Close()
	log.Debugf("Connection to tool server established")

	// TODO: move into separate function

	// Forward ports if specified
	if len(cmd.ForwardPorts) > 0 {
		return cmd.forwardPorts(ctx, toolSSHClient, log)
	}

	// Reverse forward ports if specified
	if len(cmd.ReverseForwardPorts) > 0 && !cmd.GPGAgentForwarding {
		return cmd.reverseForwardPorts(ctx, toolSSHClient, log)
	}

	// Start port-forwarding and services if enabled
	if cmd.StartServices {
		go cmd.startServices(ctx, devPodConfig, toolSSHClient, cmd.GitUsername, cmd.GitToken, wCfg, log)
	}

	log.Info("Creating user ssh client")
	// Create an SSH client for the user server
	sshClient, err := ts.WaitForSSHClient(ctx, lc, wAddr.Host(), wAddr.Port(), cmd.User, log)
	if err != nil {
		return fmt.Errorf("failed to create SSH client for user server: %w", err)
	}
	defer sshClient.Close()

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
			go startSSHKeepAlive(ctx, toolSSHClient, cmd.SSHKeepAliveInterval, log)
		}

		// TODO: What about stderr?
		// return ts.DirectTunnel(ctx, network, host, sshServer.DefaultPort, os.Stdin, os.Stdout)
		return nil
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

// TODO: When to shut down?
// FIXME: This currently only works on macOS, we'll need
// platform specific handling for this
func startDevPodD(host string) error {
	// TODO: Properly manage peros
	return nil
	name := fmt.Sprintf("sh.loft.devpodd.%s", host)
	desc := fmt.Sprintf("Daemon for DevPod Pro %s", host)
	service, err := daemon.New(name, desc, daemon.UserAgent)
	if err != nil {
		return err
	}

	args := []string{"pro", "daemon", "--host", host}
	fmt.Println(args)
	test, err := service.Install(args...)
	if err != nil {
		if errors.Is(err, daemon.ErrAlreadyInstalled) {
			return nil
		}

		return err
	}
	fmt.Println(test)

	test, err = service.Start()
	if err != nil && !errors.Is(err, daemon.ErrAlreadyRunning) {
		return err
	}
	fmt.Println(test)

	return nil
}
