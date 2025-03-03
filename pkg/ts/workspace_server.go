package ts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"

	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/provider"
	sshServer "github.com/loft-sh/devpod/pkg/ssh/server"
	"tailscale.com/client/tailscale"
	"tailscale.com/envknob"
	"tailscale.com/ipn/store"
	"tailscale.com/tsnet"
	"tailscale.com/types/netmap"
)

const (
	// TSPortForwardPort is the fixed port on which the workspace WebSocket reverse proxy listens.
	TSPortForwardPort string = "12051"
)

// WorkspaceServer holds the TSNet server and its listeners.
type WorkspaceServer struct {
	tsServer  *tsnet.Server
	listeners []net.Listener

	config *WorkspaceServerConfig
	log    log.Logger
}

// WorkspaceServerConfig defines configuration for the TSNet instance.
type WorkspaceServerConfig struct {
	AccessKey     string
	PlatformHost  string
	WorkspaceHost string
	LogF          func(format string, args ...interface{})
	Client        client.Client
	RootDir       string
}

// NewWorkspaceServer creates a new TSNet server instance.
func NewWorkspaceServer(config *WorkspaceServerConfig, logger log.Logger) *WorkspaceServer {
	return &WorkspaceServer{
		config: config,
		log:    logger,
	}
}

// Start initializes the TSNet server, sets up listeners for SSH and HTTP
// reverse proxy traffic, and waits until the given context is canceled.
func (s *WorkspaceServer) Start(ctx context.Context) error {
	s.log.Infof("Starting workspace server")

	// Perform TSNet initialization (validation, control URL, server startup, hostname parsing)
	workspaceName, projectName, err := s.setupTSNet(ctx)
	if err != nil {
		return err
	}
	lc, err := s.tsServer.LocalClient()
	if err != nil {
		return err
	}

	// Discover a runner and set up connection tracking (with heartbeat)
	discoveredRunner := s.discoverRunner(ctx, lc)
	cc := s.createConnectionCounter(ctx, discoveredRunner, projectName, workspaceName)

	// Start both SSH and HTTP reverse proxy listeners
	if err := s.startListeners(ctx, cc); err != nil {
		return err
	}

	go func() {
		WatchNetmap(ctx, lc, func(netMap *netmap.NetworkMap) {
			nm, err := json.Marshal(netMap)
			if err != nil {
				s.log.Errorf("Failed to marshal netmap: %v", err)
			} else {
				_ = os.WriteFile(filepath.Join(s.config.RootDir, "netmap.json"), nm, 0o644)
			}
		})
	}()

	// Wait until the context is canceled.
	<-ctx.Done()
	return nil
}

// Stop shuts down all listeners and the TSNet server.
func (s *WorkspaceServer) Stop() {
	for _, listener := range s.listeners {
		if listener != nil {
			listener.Close()
		}
	}
	if s.tsServer != nil {
		s.tsServer.Close()
		s.tsServer = nil
	}
	s.log.Info("Tailscale server stopped")
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

// validateConfig ensures required configuration values are set.
func (s *WorkspaceServer) validateConfig() error {
	if s.config.AccessKey == "" || s.config.PlatformHost == "" || s.config.WorkspaceHost == "" {
		return fmt.Errorf("access key, host, or hostname cannot be empty")
	}
	return nil
}

// setupControlURL constructs the control URL and verifies DERP connection.
func (s *WorkspaceServer) setupControlURL(ctx context.Context) (*url.URL, error) {
	baseURL := &url.URL{
		Scheme: GetEnvOrDefault("LOFT_TSNET_SCHEME", "https"),
		Host:   s.config.PlatformHost,
	}
	if err := CheckDerpConnection(ctx, baseURL); err != nil {
		return nil, fmt.Errorf("failed to verify DERP connection: %w", err)
	}
	return baseURL, nil
}

// initTsServer initializes the TSNet server.
func (s *WorkspaceServer) initTsServer(ctx context.Context, controlURL *url.URL) error {
	fs, err := store.NewFileStore(s.config.LogF, filepath.Join(s.config.RootDir, provider.DaemonStateFile))
	if err != nil {
		return fmt.Errorf("failed to create file store: %w", err)
	}
	envknob.Setenv("TS_DEBUG_TLS_DIAL_INSECURE_SKIP_VERIFY", "true")
	s.log.Infof("Connecting to control URL - %s/coordinator/", controlURL.String())
	s.tsServer = &tsnet.Server{
		Hostname:   s.config.WorkspaceHost,
		Logf:       s.config.LogF,
		ControlURL: controlURL.String() + "/coordinator/",
		AuthKey:    s.config.AccessKey,
		Dir:        s.config.RootDir,
		Ephemeral:  false,
		Store:      fs,
	}
	if _, err := s.tsServer.Up(ctx); err != nil {
		return fmt.Errorf("failed to start tsnet server: %w", err)
	}
	return nil
}

// parseHostname extracts workspace and project names from the hostname.
func (s *WorkspaceServer) parseWorkspaceHostname() (workspace, project string, err error) {
	parts := strings.Split(s.config.WorkspaceHost, ".")
	if len(parts) < 4 {
		return "", "", fmt.Errorf("invalid workspace hostname format: %s", s.config.WorkspaceHost)
	}
	return parts[1], parts[2], nil
}

// startListeners creates and starts the SSH and HTTP reverse proxy listeners.
func (s *WorkspaceServer) startListeners(ctx context.Context, cc *connectionCounter) error {
	// Create and start the SSH listener.
	s.log.Infof("Starting SSH listener")
	sshListener, err := s.createListener(ctx, fmt.Sprintf(":%d", sshServer.DefaultUserPort), cc)
	if err != nil {
		return err
	}

	// Create and start the HTTP reverse proxy listener.
	s.log.Infof("Starting HTTP reverse proxy listener on TSNet port %s", TSPortForwardPort)
	wsListener, err := s.createListener(ctx, fmt.Sprintf(":%s", TSPortForwardPort), cc)
	if err != nil {
		return fmt.Errorf("failed to create listener on TS port %s: %w", TSPortForwardPort, err)
	}

	s.listeners = append(s.listeners, sshListener, wsListener)

	// Setup HTTP handler for port forwarding.
	mux := http.NewServeMux()
	mux.HandleFunc("/portforward", s.httpPortForwardHandler)

	go func() {
		if err := http.Serve(wsListener, mux); err != nil && err != http.ErrServerClosed {
			s.log.Errorf("HTTP server error on TS port %s: %v", TSPortForwardPort, err)
		}
	}()

	// Start handling SSH connections.
	go s.handleSSHConnections(ctx, sshListener)
	return nil
}

// createListener creates a raw listener and wraps it with connection tracking.
func (s *WorkspaceServer) createListener(ctx context.Context, addr string, cc *connectionCounter) (net.Listener, error) {
	l, err := s.tsServer.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	return newTrackedListener(l, cc), nil
}

// httpPortForwardHandler is the HTTP reverse proxy handler for workspace.
// It reconstructs the target URL using custom headers and forwards the request.
func (s *WorkspaceServer) httpPortForwardHandler(w http.ResponseWriter, r *http.Request) {
	s.log.Debugf("httpPortForwardHandler: starting")

	// Retrieve required custom headers.
	targetPort := r.Header.Get("X-Loft-Forward-Port")
	baseForwardStr := r.Header.Get("X-Loft-Forward-Url")
	if targetPort == "" || baseForwardStr == "" {
		http.Error(w, "missing required X-Loft headers", http.StatusBadRequest)
		return
	}
	s.log.Debugf("httpPortForwardHandler: received headers: X-Loft-Forward-Port=%s, X-Loft-Forward-Url=%s", targetPort, baseForwardStr)

	// Parse and modify the URL to target the local endpoint.
	parsedURL, err := url.Parse(baseForwardStr)
	if err != nil {
		s.log.Errorf("httpPortForwardHandler: failed to parse base URL: %v", err)
		http.Error(w, "invalid base forward URL", http.StatusBadRequest)
		return
	}
	parsedURL.Scheme = "http"
	parsedURL.Host = "127.0.0.1:" + targetPort
	s.log.Debugf("httpPortForwardHandler: final target URL=%s", parsedURL.String())

	// Build the reverse proxy with a custom Director.
	proxy := httputil.NewSingleHostReverseProxy(parsedURL)
	proxy.Director = func(req *http.Request) {
		dest := *parsedURL
		req.URL = &dest
		req.Host = dest.Host
		// Remove custom headers so they are not forwarded.
		req.Header.Del("X-Loft-Forward-Port")
		req.Header.Del("X-Loft-Forward-Url")
		req.Header.Del("X-Loft-Forward-Authorization")
	}
	proxy.Transport = http.DefaultTransport

	s.log.Infof("httpPortForwardHandler: final proxied request: %s %s", r.Method, parsedURL.String())
	proxy.ServeHTTP(w, r)
}

// handleSSHConnections continuously accepts SSH connections and handles each one.
func (s *WorkspaceServer) handleSSHConnections(ctx context.Context, listener net.Listener) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		clientConn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			s.log.Errorf("Failed to accept connection: %v", err)
			continue
		}
		clientHost, _, err := net.SplitHostPort(clientConn.RemoteAddr().String())
		if err != nil {
			s.log.Infof("Unable to parse host: %s", clientConn.RemoteAddr().String())
			clientConn.Close()
			continue
		}
		go s.handleSSHConnection(clientConn, clientHost)
	}
}

// handleSSHConnection proxies the SSH connection to the local backend.
func (s *WorkspaceServer) handleSSHConnection(clientConn net.Conn, clientHost string) {
	defer clientConn.Close()

	localAddr := fmt.Sprintf("127.0.0.1:%d", sshServer.DefaultUserPort)
	backendConn, err := net.Dial("tcp", localAddr)
	if err != nil {
		s.log.Errorf("Failed to connect to local address %s: %v", localAddr, err)
		return
	}
	defer backendConn.Close()

	// Start bidirectional copy between client and backend.
	go func() {
		defer clientConn.Close()
		defer backendConn.Close()
		_, err = io.Copy(backendConn, clientConn)
	}()
	_, err = io.Copy(clientConn, backendConn)
}

// discoverRunner attempts to find the runner peer from the TSNet status.
func (s *WorkspaceServer) discoverRunner(ctx context.Context, lc *tailscale.LocalClient) string {
	status, err := lc.Status(ctx)
	if err != nil {
		s.log.Infof("discoverRunner: failed to get status: %v", err)
		return ""
	}
	var runner string
	for _, peer := range status.Peer {
		if peer == nil || peer.HostName == "" {
			continue
		}
		s.log.Infof("discoverRunner: found peer %s with IPs %v", peer.HostName, peer.TailscaleIPs)
		if strings.HasSuffix(peer.HostName, "runner") {
			runner = peer.HostName
			break
		}
	}
	s.log.Infof("discoverRunner: selected runner = %s", runner)
	return runner
}

// createConnectionCounter sets up a connection counter with heartbeat callbacks.
func (s *WorkspaceServer) createConnectionCounter(ctx context.Context, discoveredRunner, projectName, workspaceName string) *connectionCounter {
	var (
		heartbeatMu sync.Mutex
		heartbeats  = make(map[string]context.CancelFunc)
	)
	onConnect := func(address string) {
		s.log.Infof("Client %s connected", address)
		heartbeatMu.Lock()
		if _, exists := heartbeats[address]; exists {
			heartbeatMu.Unlock()
			return
		}
		hbCtx, cancel := context.WithCancel(ctx)
		heartbeats[address] = cancel
		heartbeatMu.Unlock()
		heartbeatURL := fmt.Sprintf("http://%s.ts.loft/devpod/%s/%s/heartbeat", discoveredRunner, projectName, workspaceName)
		s.log.Debugf("Setting up heartbeat for %s with URL %s", address, heartbeatURL)
		transport := &http.Transport{DialContext: s.tsServer.Dial}
		client := &http.Client{Transport: transport, Timeout: 10 * time.Second}
		go func(addr string, hbCtx context.Context) {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-hbCtx.Done():
					s.log.Infof("Heartbeat for client %s stopped", address)
					return
				case <-ticker.C:
					req, err := http.NewRequestWithContext(hbCtx, "GET", heartbeatURL, nil)
					s.log.Infof("Building request")
					if err != nil {
						s.log.Infof("Heartbeat: failed to create request for %s: %v", heartbeatURL, err)
						continue
					}
					req.Header.Set("Authorization", "Bearer "+s.config.AccessKey)
					resp, err := client.Do(req)
					if err != nil {
						s.log.Infof("Heartbeat: request to %s failed: %v", heartbeatURL, err)
						continue
					}
					s.log.Infof("Heartbeat: received response from %s - Status: %d", heartbeatURL, resp.StatusCode)
					resp.Body.Close()
				}
			}
		}(address, hbCtx)
	}
	onDisconnect := func(address string) {
		s.log.Infof("Client %s disconnected", address)
		heartbeatMu.Lock()
		if cancel, exists := heartbeats[address]; exists {
			cancel()
			delete(heartbeats, address)
		}
		heartbeatMu.Unlock()
	}
	gracePeriod := 5 * time.Second
	return newConnectionCounter(ctx, gracePeriod, onConnect, onDisconnect, log.Default.WithLevel(logrus.DebugLevel))
}

// trackedListener wraps a net.Listener to track connections.
type trackedListener struct {
	net.Listener
	cc *connectionCounter
}

func newTrackedListener(l net.Listener, cc *connectionCounter) net.Listener {
	return &trackedListener{
		Listener: l,
		cc:       cc,
	}
}

func (tl *trackedListener) Accept() (net.Conn, error) {
	conn, err := tl.Listener.Accept()
	if err != nil {
		return nil, err
	}
	remote := conn.RemoteAddr().String()
	tl.cc.Add(remote)
	return newTrackedConn(conn, tl.cc, remote), nil
}

// trackedConn wraps a net.Conn to ensure the connection counter is updated when closing.
type trackedConn struct {
	net.Conn
	cc     *connectionCounter
	remote string
	once   sync.Once
}

func newTrackedConn(c net.Conn, cc *connectionCounter, remote string) net.Conn {
	return &trackedConn{
		Conn:   c,
		cc:     cc,
		remote: remote,
	}
}

func (tc *trackedConn) Close() error {
	var err error
	tc.once.Do(func() {
		err = tc.Conn.Close()
		tc.cc.Dec(tc.remote)
	})
	return err
}
