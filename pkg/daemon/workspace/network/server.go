package network

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	"tailscale.com/envknob"
	"tailscale.com/ipn/store/mem"
	"tailscale.com/tsnet"
)

type WorkspaceServerConfig struct {
	AccessKey     string
	PlatformHost  string
	WorkspaceHost string
	LogF          func(format string, args ...interface{})
	Client        client.Client
	RootDir       string
}

// WorkspaceServer is the main workspace network server.
// It creates and manages network server instance as well as
// all services that run on DevPod network inside the workspace.
type WorkspaceServer struct {
	network     *tsnet.Server // TODO: we probably want to hide network behind our own interface at some point
	config      *WorkspaceServerConfig
	log         log.Logger
	connTracker *ConnTracker

	// Services
	sshSvc                    *SSHService
	httpProxySvc              *HTTPPortForwardService
	platformGitCredentialsSvc *PlatformGitCredentialsService
	netProxySvc               *NetworkProxyService
	heartbeatSvc              *HeartbeatService
	netmapWatcher             *NetmapWatcherService
}

// NewWorkspaceServer creates a new WorkspaceServer.
func NewWorkspaceServer(config *WorkspaceServerConfig, logger log.Logger) *WorkspaceServer {
	return &WorkspaceServer{
		config:      config,
		log:         logger,
		connTracker: &ConnTracker{},
	}
}

// Start initializes the network server server and all services, then blocks until the context is canceled.
func (s *WorkspaceServer) Start(ctx context.Context) error {
	s.log.Infof("Starting workspace server")
	workspaceName, projectName, err := s.joinNetwork(ctx)
	if err != nil {
		return err
	}

	lc, err := s.network.LocalClient()
	if err != nil {
		return err
	}

	// Create and start the SSH service.
	s.sshSvc, err = NewSSHService(s.network, s.connTracker, s.log)
	if err != nil {
		return err
	}
	s.sshSvc.Start(ctx)

	// Create and start the HTTP port forward service.
	s.httpProxySvc, err = NewHTTPPortForwardService(s.network, s.connTracker, s.log)
	if err != nil {
		return err
	}
	s.httpProxySvc.Start(ctx)

	// Create and start the platform git credentials service.
	s.platformGitCredentialsSvc, err = NewPlatformGitCredentialsService(s.config, s.network, lc, projectName, workspaceName, s.log)
	if err != nil {
		return err
	}
	s.platformGitCredentialsSvc.Start(ctx)

	// Create and start the network proxy service.
	networkSocket := filepath.Join(s.config.RootDir, NetworkProxySocket)
	s.netProxySvc, err = NewNetworkProxyService(networkSocket, s.network, s.log)
	if err != nil {
		return err
	}
	s.netProxySvc.Start(ctx)

	// Start the heartbeat service.
	s.heartbeatSvc = NewHeartbeatService(s.config, s.network, lc, projectName, workspaceName, s.connTracker, s.log)
	go s.heartbeatSvc.Start(ctx)

	// Start netmap watcher.
	s.netmapWatcher = NewNetmapWatcherService(s.config.RootDir, lc, s.log)
	s.netmapWatcher.Start(ctx)

	// Wait until the context is canceled.
	<-ctx.Done()
	return nil
}

// Stop shuts down all services and the network server.
func (s *WorkspaceServer) Stop() {
	if s.sshSvc != nil {
		s.sshSvc.Stop()
	}
	if s.httpProxySvc != nil {
		s.httpProxySvc.Stop()
	}
	if s.platformGitCredentialsSvc != nil {
		s.platformGitCredentialsSvc.Stop()
	}
	if s.netProxySvc != nil {
		s.netProxySvc.Stop()
	}
	if s.network != nil {
		s.network.Close()
		s.network = nil
	}
	s.log.Info("Workspace server stopped")
}

// Dial dials the given address using the network server.
func (s *WorkspaceServer) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	if s.network == nil {
		return nil, fmt.Errorf("network server is not running")
	}
	return s.network.Dial(ctx, network, addr)
}

// joinNetwork validates configuration, sets up the control URL, starts the network server,
// and parses the hostname into workspace and project names.
func (s *WorkspaceServer) joinNetwork(ctx context.Context) (workspace, project string, err error) {
	if err = s.validateConfig(); err != nil {
		return "", "", err
	}
	baseURL, err := s.setupControlURL(ctx)
	if err != nil {
		return "", "", err
	}
	if err = s.initNetworkServer(ctx, baseURL); err != nil {
		return "", "", err
	}
	return s.parseWorkspaceHostname()
}

func (s *WorkspaceServer) validateConfig() error {
	if s.config.AccessKey == "" || s.config.PlatformHost == "" || s.config.WorkspaceHost == "" {
		return fmt.Errorf("access key, host, or hostname cannot be empty")
	}
	return nil
}

func (s *WorkspaceServer) setupControlURL(ctx context.Context) (*url.URL, error) {
	baseURL := &url.URL{
		Scheme: ts.GetEnvOrDefault("LOFT_TSNET_SCHEME", "https"),
		Host:   s.config.PlatformHost,
	}
	if err := ts.CheckDerpConnection(ctx, baseURL); err != nil {
		return nil, fmt.Errorf("failed to verify DERP connection: %w", err)
	}
	return baseURL, nil
}

func (s *WorkspaceServer) initNetworkServer(ctx context.Context, controlURL *url.URL) error {
	store, _ := mem.New(s.config.LogF, "")
	envknob.Setenv("TS_DEBUG_TLS_DIAL_INSECURE_SKIP_VERIFY", "true")
	s.log.Infof("Connecting to control URL - %s/coordinator/", controlURL.String())
	s.network = &tsnet.Server{ // TODO: this probably could be extracted from here and local daemon into pkg/ts
		Hostname:   s.config.WorkspaceHost,
		Logf:       s.config.LogF,
		ControlURL: controlURL.String() + "/coordinator/",
		AuthKey:    s.config.AccessKey,
		Dir:        s.config.RootDir,
		Ephemeral:  true,
		Store:      store,
	}
	if _, err := s.network.Up(ctx); err != nil {
		return err
	}
	return nil
}

func (s *WorkspaceServer) parseWorkspaceHostname() (workspace, project string, err error) {
	parts := strings.Split(s.config.WorkspaceHost, ".")
	if len(parts) < 4 {
		return "", "", fmt.Errorf("invalid workspace hostname format: %s", s.config.WorkspaceHost)
	}
	return parts[1], parts[2], nil
}
