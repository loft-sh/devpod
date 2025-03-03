package daemonclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
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
	daemonDir, err := provider.GetDaemonDir(devPodConfig.DefaultContext, workspace.Provider.Name)
	if err != nil {
		return nil, err
	}
	tsClient := &tailscale.LocalClient{
		Socket:        daemon.GetSocketAddr(daemonDir, workspace.Provider.Name),
		UseSocketOnly: true,
	}
	localClient := daemon.NewLocalClient(daemonDir, prov.Name)

	return &client{
		devPodConfig: devPodConfig,
		config:       prov,
		workspace:    workspace,
		log:          log,
		tsClient:     tsClient,
		localClient:  localClient,
	}, nil
}

type client struct {
	m sync.Mutex

	workspaceLockOnce sync.Once
	workspaceLock     *flock.Flock

	devPodConfig *config.Config
	config       *provider.ProviderConfig
	workspace    *provider.Workspace
	log          log.Logger
	tsClient     *tailscale.LocalClient
	localClient  *daemon.LocalClient
}

func (c *client) Lock(ctx context.Context) error {
	c.initLock()

	// try to lock workspace
	c.log.Debugf("Acquire workspace lock...")
	err := tryLock(ctx, c.workspaceLock, "workspace", c.log)
	if err != nil {
		return fmt.Errorf("error locking workspace: %w", err)
	}
	c.log.Debugf("Acquired workspace lock...")

	return nil
}

func (c *client) Unlock() {
	c.initLock()

	// try to unlock workspace
	err := c.workspaceLock.Unlock()
	if err != nil {
		c.log.Warnf("Error unlocking workspace: %v", err)
	}
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

func (c *client) SSHClients(ctx context.Context, user string) (toolClient *ssh.Client, userClient *ssh.Client, err error) {
	wAddr, err := c.getWorkspaceAddress()
	if err != nil {
		return nil, nil, fmt.Errorf("resolve workspace hostname: %w", err)
	}
	err = ts.WaitHostReachable(ctx, c.tsClient, wAddr, c.log)
	if err != nil {
		return nil, nil, fmt.Errorf("reach host: %w", err)
	}

	c.log.Debugf("Host %s is reachable. Proceeding with SSH session...", wAddr.Host())

	toolClient, err = ts.WaitForSSHClient(ctx, c.tsClient, wAddr.Host(), wAddr.Port(), "root", c.log)
	if err != nil {
		return nil, nil, fmt.Errorf("create SSH tool client: %w", err)
	}
	userClient, err = ts.WaitForSSHClient(ctx, c.tsClient, wAddr.Host(), wAddr.Port(), user, c.log)
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

func (c *client) initLock() {
	c.workspaceLockOnce.Do(func() {
		c.m.Lock()
		defer c.m.Unlock()

		// get locks dir
		workspaceLocksDir, err := provider.GetLocksDir(c.workspace.Context)
		if err != nil {
			panic(fmt.Errorf("get workspaces dir: %w", err))
		}
		_ = os.MkdirAll(workspaceLocksDir, 0777)

		// create workspace lock
		c.workspaceLock = flock.New(filepath.Join(workspaceLocksDir, c.workspace.ID+".workspace.lock"))
	})
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
		writer.Write([]byte(fmt.Sprintf("pong from %s (%s) via %v in %v\n", result.NodeName, result.NodeIP, via, latency)))

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
	if c.workspace.Pro == nil {
		return ts.Addr{}, fmt.Errorf("workspace is not initialized")
	}

	return ts.NewAddr(ts.GetWorkspaceHostname(c.workspace.Pro.InstanceName, c.workspace.Pro.Project), sshServer.DefaultUserPort), nil
}

func printLogMessagePeriodically(message string, log log.Logger) chan struct{} {
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(time.Second * 5):
				log.Info(message)
			}
		}
	}()

	return done
}

func tryLock(ctx context.Context, lock *flock.Flock, name string, log log.Logger) error {
	done := printLogMessagePeriodically(fmt.Sprintf("Trying to lock %s, seems like another process is running that blocks this %s", name, name), log)
	defer close(done)

	now := time.Now()
	for time.Since(now) < time.Minute*5 {
		locked, err := lock.TryLock()
		if err != nil {
			return err
		} else if locked {
			return nil
		}

		select {
		case <-time.After(time.Second):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("timed out waiting to lock %s, seems like there is another process running on this machine that blocks it", name)
}
