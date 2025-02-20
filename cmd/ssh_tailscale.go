package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/platform/daemon"
	"github.com/loft-sh/devpod/pkg/provider"
	"os"

	"github.com/loft-sh/devpod/cmd/machine"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/gpg"
	sshServer "github.com/loft-sh/devpod/pkg/ssh/server"
	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	tsclient "tailscale.com/client/tailscale"
)

func startTSProxyTunnel(
	ctx context.Context,
	devPodConfig *config.Config,
	client client2.ProxyClient,
	cmd SSHCmd,
	log log.Logger,
) error {
	log.Debugf("Starting proxy connection")

	providerName := client.WorkspaceConfig().Provider.Name
	tsDir, err := provider.GetDaemonDir(devPodConfig.DefaultContext, providerName)
	if err != nil {
		return err
	}
	lc := &tsclient.LocalClient{
		Socket:        daemon.GetSocketAddr(tsDir, providerName),
		UseSocketOnly: true,
	}
	status, err := lc.Status(ctx)
	if err != nil {
		return fmt.Errorf("connect to daemon: %w", err)
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

	wCfg := client.WorkspaceConfig()
	wAddr := ts.NewAddr(ts.GetWorkspaceHostname(wCfg.Pro.InstanceName, wCfg.Pro.Project), sshServer.DefaultUserPort)

	err = ts.WaitHostReachable(ctx, lc, wAddr, log)
	if err != nil {
		return fmt.Errorf("failed to reach TSNet host: %w", err)
	}

	log.Debugf("Host %s is reachable. Proceeding with SSH session...", wAddr.Host())

	// Root ssh client for tool
	toolSSHClient, err := ts.WaitForSSHClient(ctx, lc, wAddr.Host(), wAddr.Port(), "root", log)
	if err != nil {
		return fmt.Errorf("failed to create SSH client for tool server: %w", err)
	}
	defer toolSSHClient.Close()

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

		return ts.DirectTunnel(ctx, lc, wAddr.Host(), wAddr.Port(), os.Stdin, os.Stdout)
	}

	// User ssh client for actual session
	sshClient, err := ts.WaitForSSHClient(ctx, lc, wAddr.Host(), wAddr.Port(), cmd.User, log)
	if err != nil {
		return fmt.Errorf("failed to create SSH client for user server: %w", err)
	}

	defer sshClient.Close()
	// Connect to the inner server and handle user session
	return machine.RunSSHSession(
		ctx,
		sshClient,
		cmd.AgentForwarding,
		cmd.Command,
		os.Stderr,
	)
}
