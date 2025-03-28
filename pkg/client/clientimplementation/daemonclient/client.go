package daemonclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	clientpkg "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	daemon "github.com/loft-sh/devpod/pkg/daemon/platform"
	"github.com/loft-sh/devpod/pkg/options"
	"github.com/loft-sh/devpod/pkg/platform"
	platformclient "github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/provider"
	sshServer "github.com/loft-sh/devpod/pkg/ssh/server"
	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	perrors "github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
	"golang.org/x/crypto/ssh"
	"tailscale.com/client/tailscale"
	"tailscale.com/tailcfg"
)

var (
	DevPodDebug = "DEVPOD_DEBUG"

	DevPodFlagsUp     = "DEVPOD_FLAGS_UP"
	DevPodFlagsSsh    = "DEVPOD_FLAGS_SSH"
	DevPodFlagsDelete = "DEVPOD_FLAGS_DELETE"
	DevPodFlagsStatus = "DEVPOD_FLAGS_STATUS"
)

func New(devPodConfig *config.Config, prov *provider.ProviderConfig, workspace *provider.Workspace, log log.Logger) (clientpkg.DaemonClient, error) {
	tsClient := &tailscale.LocalClient{
		Socket:        daemon.GetSocketAddr(workspace.Provider.Name),
		UseSocketOnly: true,
	}

	return &client{
		devPodConfig: devPodConfig,
		config:       prov,
		workspace:    workspace,
		log:          log,
		tsClient:     tsClient,
		localClient:  daemon.NewLocalClient(prov.Name),
	}, nil
}

type client struct {
	m sync.Mutex

	devPodConfig *config.Config
	config       *provider.ProviderConfig
	workspace    *provider.Workspace
	log          log.Logger
	tsClient     *tailscale.LocalClient
	localClient  *daemon.LocalClient
}

func (c *client) Lock(ctx context.Context) error {
	// noop
	return nil
}

func (c *client) Unlock() {
	// noop
}

func (c *client) Provider() string {
	return c.config.Name
}

func (c *client) Workspace() string {
	c.m.Lock()
	defer c.m.Unlock()

	return c.workspace.ID
}

func (c *client) WorkspaceConfig() *provider.Workspace {
	c.m.Lock()
	defer c.m.Unlock()

	return provider.CloneWorkspace(c.workspace)
}

func (c *client) Context() string {
	return c.workspace.Context
}

func (c *client) RefreshOptions(ctx context.Context, userOptionsRaw []string, reconfigure bool) error {
	c.m.Lock()
	defer c.m.Unlock()

	userOptions, err := provider.ParseOptions(userOptionsRaw)
	if err != nil {
		return perrors.Wrap(err, "parse options")
	}

	workspace, err := options.ResolveAndSaveOptionsProxy(ctx, c.devPodConfig, c.config, c.workspace, userOptions, c.log)
	if err != nil {
		return err
	}

	if reconfigure {
		err := c.updateInstance(ctx)
		if err != nil {
			return err
		}
	}

	c.workspace = workspace
	return nil
}

func (c *client) CheckWorkspaceReachable(ctx context.Context) error {
	wAddr, err := c.getWorkspaceAddress()
	if err != nil {
		return fmt.Errorf("resolve workspace hostname: %w", err)
	}
	err = ts.WaitHostReachable(ctx, c.tsClient, wAddr, 5, c.log)
	if err != nil {
		instance, getWorkspaceErr := c.localClient.GetWorkspace(ctx, c.workspace.UID)
		// if we can't reach the daemon try to start the desktop app
		if daemon.IsDaemonNotAvailableError(getWorkspaceErr) {
			deeplink := fmt.Sprintf("devpod://open?workspace=%s&provider=%s&source=%s&ide=%s", c.workspace.ID, c.config.Name, c.workspace.Source.String(), c.workspace.IDE.Name)
			openErr := open.Run(deeplink)
			if openErr != nil {
				return getWorkspaceErr // inform user about daemon state
			}
			// give desktop app a chance to start
			time.Sleep(2 * time.Second)

			// let's try again
			err = ts.WaitHostReachable(ctx, c.tsClient, wAddr, 20, c.log)
			if err != nil {
				instance, getWorkspaceErr = c.localClient.GetWorkspace(ctx, c.workspace.UID)
			} else {
				return nil
			}
		}

		if getWorkspaceErr != nil {
			return fmt.Errorf("couldn't get workspace: %w", getWorkspaceErr)
		} else if instance.Status.Phase != storagev1.InstanceReady {
			return fmt.Errorf("workspace is '%s', please run `devpod up %s` to start it again", instance.Status.Phase, c.workspace.ID)
		} else if instance.Status.LastWorkspaceStatus != storagev1.WorkspaceStatusRunning {
			return fmt.Errorf("workspace is '%s', please run `devpod up %s` to start it again", instance.Status.LastWorkspaceStatus, c.workspace.ID)
		}

		return fmt.Errorf("reach host: %w", err)
	}

	c.log.Debugf("Host %s is reachable. Proceeding with SSH session...", wAddr.Host())
	return nil
}

func (c *client) SSHClients(ctx context.Context, user string) (toolClient *ssh.Client, userClient *ssh.Client, err error) {
	wAddr, err := c.getWorkspaceAddress()
	if err != nil {
		return nil, nil, fmt.Errorf("resolve workspace hostname: %w", err)
	}

	address := fmt.Sprintf("%s:%d", wAddr.Host(), wAddr.Port())
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		addressParts := strings.Split(address, ":")
		if len(addressParts) != 2 {
			return nil, fmt.Errorf("invalid address: %s", address)
		}

		port, err := strconv.Atoi(addressParts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid port: %s", addressParts[1])
		}

		return c.tsClient.DialTCP(ctx, addressParts[0], uint16(port))
	}

	toolClient, err = ts.WaitForSSHClient(ctx, dial, "tcp", address, "root", time.Second*10, c.log)
	if err != nil {
		return nil, nil, fmt.Errorf("create SSH tool client: %w", err)
	}
	userClient, err = ts.WaitForSSHClient(ctx, dial, "tcp", address, user, time.Second*10, c.log)
	if err != nil {
		return nil, nil, fmt.Errorf("create SSH user client: %w", err)
	}

	return toolClient, userClient, nil
}

func (c *client) DirectTunnel(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	wAddr, err := c.getWorkspaceAddress()
	if err != nil {
		return fmt.Errorf("resolve workspace hostname: %w", err)
	}
	conn, err := c.tsClient.DialTCP(ctx, wAddr.Host(), uint16(wAddr.Port()))
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server in proxy mode: %w", err)
	}
	defer conn.Close()

	errChan := make(chan error, 1)
	go func() {
		_, err := io.Copy(stdout, conn)
		errChan <- err
	}()
	go func() {
		_, err := io.Copy(conn, stdin)
		errChan <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

func (c *client) Ping(ctx context.Context, writer io.Writer) error {
	wAddr, err := c.getWorkspaceAddress()
	if err != nil {
		return err
	}
	status, err := c.tsClient.Status(ctx)
	if err != nil {
		return err
	}
	hostname := strings.TrimSuffix(wAddr.Host(), "."+status.CurrentTailnet.Name)
	var ip *netip.Addr
	for _, peer := range status.Peer {
		if peer.HostName == hostname {
			ip = &peer.TailscaleIPs[0]
		}
	}

	if ip == nil {
		return fmt.Errorf("no network peer for hostname %s", wAddr.Host())
	}

	for i := 0; i < 10; i++ {
		timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		result, err := c.tsClient.Ping(timeoutCtx, *ip, tailcfg.PingDisco)
		if err != nil {
			return err
		}
		if result.Err != "" {
			return errors.New(result.Err)
		}
		latency := time.Duration(result.LatencySeconds * float64(time.Second)).Round(time.Millisecond)
		via := result.Endpoint
		if result.DERPRegionID != 0 {
			via = fmt.Sprintf("DERP(%s)", result.DERPRegionCode)
		}
		_, err = writer.Write([]byte(fmt.Sprintf("pong from %s (%s) via %v in %v\n", result.NodeName, result.NodeIP, via, latency)))
		if err != nil {
			return fmt.Errorf("failed to write ping result: %w", err)
		}

		time.Sleep(time.Second)
	}

	return nil
}

func (c *client) initPlatformClient(ctx context.Context) (platformclient.Client, error) {
	configPath, err := platform.LoftConfigPath(c.Context(), c.Provider())
	if err != nil {
		return nil, err
	}
	baseClient, err := platformclient.InitClientFromPath(ctx, configPath)
	if err != nil {
		return nil, err
	}

	return baseClient, nil
}

func (c *client) getWorkspaceAddress() (ts.Addr, error) {
	if c.workspace.Pro == nil || c.workspace.Pro.InstanceName == "" {
		return ts.Addr{}, fmt.Errorf("workspace is not initialized")
	}

	return ts.NewAddr(ts.GetWorkspaceHostname(c.workspace.Pro.InstanceName, c.workspace.Pro.Project), sshServer.DefaultUserPort), nil
}
