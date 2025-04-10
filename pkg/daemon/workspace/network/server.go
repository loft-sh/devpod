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
	tsServer    *tsnet.Server
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
	workspaceName, projectName, err := s.setupTSNet(ctx)
	if err != nil {
		return err
	}

	lc, err := s.tsServer.LocalClient()
	if err != nil {
		return err
	}

	// Create and start the SSH service.
	s.sshSvc, err = NewSSHServer(s.tsServer, s.connTracker, s.log)
	if err != nil {
		return err
	}
	s.sshSvc.Start(ctx)

	// Create and start the HTTP port forward service.
	s.httpProxySvc, err = NewHTTPPortForwardService(s.tsServer, s.connTracker, s.log)
	if err != nil {
		return err
	}
	s.httpProxySvc.Start(ctx)

	// Create and start the platform git credentials service.
	s.platformGitCredentialsSvc, err = NewPlatformGitCredentialsService(s.config, s.tsServer, lc, projectName, workspaceName, s.log)
	if err != nil {
		return err
	}
	s.platformGitCredentialsSvc.Start(ctx)

	// Create and start the TS proxy service.
	tsProxySocket := filepath.Join(s.config.RootDir, TSNetProxySocket)
	s.netProxySvc, err = NewNetworkProxyService(tsProxySocket, s.tsServer, s.log)
	if err != nil {
		return err
	}
	s.netProxySvc.Start(ctx)

	// Start the heartbeat service.
	s.heartbeatSvc = NewHeartbeatService(s.config, s.tsServer, lc, projectName, workspaceName, s.connTracker, s.log)
	go s.heartbeatSvc.Start(ctx)

	// Start netmap watcher.
	s.netmapWatcher = NewNetmapWatcherService(s.config.RootDir, lc, s.log)
	s.netmapWatcher.Start(ctx)

	// Wait until the context is canceled.
	<-ctx.Done()
	return nil
}

// Stop shuts down all sub-servers and the TSNet server.
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
	if s.tsServer != nil {
		s.tsServer.Close()
		s.tsServer = nil
	}
	s.log.Info("Workspace server stopped")
}

// Dial dials the given address using the TSNet server.
func (s *WorkspaceServer) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	if s.tsServer == nil {
		return nil, fmt.Errorf("tailscale server is not running")
	}
	return s.tsServer.Dial(ctx, network, addr)
}

// setupTSNet validates configuration, sets up the control URL, starts the TSNet server,
// and parses the hostname into workspace and project names.
func (s *WorkspaceServer) setupTSNet(ctx context.Context) (workspace, project string, err error) {
	if err = s.validateConfig(); err != nil {
		return "", "", err
	}
	baseURL, err := s.setupControlURL(ctx)
	if err != nil {
		return "", "", err
	}
	if err = s.initTsServer(ctx, baseURL); err != nil {
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

func (s *WorkspaceServer) initTsServer(ctx context.Context, controlURL *url.URL) error {
	store, _ := mem.New(s.config.LogF, "")
	envknob.Setenv("TS_DEBUG_TLS_DIAL_INSECURE_SKIP_VERIFY", "true")
	s.log.Infof("Connecting to control URL - %s/coordinator/", controlURL.String())
	s.tsServer = &tsnet.Server{
		Hostname:   s.config.WorkspaceHost,
		Logf:       s.config.LogF,
		ControlURL: controlURL.String() + "/coordinator/",
		AuthKey:    s.config.AccessKey,
		Dir:        s.config.RootDir,
		Ephemeral:  true,
		Store:      store,
	}
	if _, err := s.tsServer.Up(ctx); err != nil {
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
