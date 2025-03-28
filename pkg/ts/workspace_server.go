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

	"github.com/loft-sh/devpod/pkg/platform/client"
	sshServer "github.com/loft-sh/devpod/pkg/ssh/server"
	"tailscale.com/client/tailscale"
	"tailscale.com/envknob"
	"tailscale.com/ipn/store/mem"
	"tailscale.com/tsnet"
	"tailscale.com/types/netmap"
)

const (
	// TSPortForwardPort is the fixed port on which the workspace WebSocket reverse proxy listens.
	TSPortForwardPort string = "12051"

	RunnerProxySocket string = "runner-proxy.sock"

	netMapCooldown = 30 * time.Second
)

// WorkspaceServer holds the TSNet server and its listeners.
type WorkspaceServer struct {
	tsServer  *tsnet.Server
	listeners []net.Listener

	connectionCounter   int
	connectionCounterMu sync.Mutex

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

	// send heartbeats
	go s.sendHeartbeats(ctx, projectName, workspaceName, lc)

	// Start both SSH and HTTP reverse proxy listeners
	if err := s.startListeners(ctx, projectName, workspaceName, lc); err != nil {
		return err
	}

	go func() {
		lastUpdate := time.Now()
		if err := WatchNetmap(ctx, lc, func(netMap *netmap.NetworkMap) {
			if time.Since(lastUpdate) < netMapCooldown {
				return
			}
			lastUpdate = time.Now()

			nm, err := json.Marshal(netMap)
			if err != nil {
				s.log.Errorf("Failed to marshal netmap: %v", err)
			} else {
				_ = os.WriteFile(filepath.Join(s.config.RootDir, "netmap.json"), nm, 0o644)
			}
		}); err != nil {
			s.log.Errorf("Failed to watch netmap: %v", err)
		}
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

// parseHostname extracts workspace and project names from the hostname.
func (s *WorkspaceServer) parseWorkspaceHostname() (workspace, project string, err error) {
	parts := strings.Split(s.config.WorkspaceHost, ".")
	if len(parts) < 4 {
		return "", "", fmt.Errorf("invalid workspace hostname format: %s", s.config.WorkspaceHost)
	}
	return parts[1], parts[2], nil
}

// startListeners creates and starts the SSH and HTTP reverse proxy listeners.
func (s *WorkspaceServer) startListeners(ctx context.Context, projectName, workspaceName string, lc *tailscale.LocalClient) error {
	// Create and start the SSH listener.
	s.log.Infof("Starting SSH listener")
	sshListener, err := s.createListener(fmt.Sprintf(":%d", sshServer.DefaultUserPort))
	if err != nil {
		return err
	}

	// Create and start the HTTP reverse proxy listener.
	s.log.Infof("Starting HTTP reverse proxy listener on TSNet port %s", TSPortForwardPort)
	wsListener, err := s.createListener(fmt.Sprintf(":%s", TSPortForwardPort))
	if err != nil {
		return fmt.Errorf("failed to create listener on TS port %s: %w", TSPortForwardPort, err)
	}

	// Create and start the platform HTTP git credentials listener
	runnerProxySocket := filepath.Join(s.config.RootDir, RunnerProxySocket)
	s.log.Infof("Starting runner proxy socket on %s", runnerProxySocket)
	_ = os.Remove(runnerProxySocket)
	runnerProxyListener, err := net.Listen("unix", runnerProxySocket)
	if err != nil {
		return fmt.Errorf("failed to create listener on TS port %s: %w", TSPortForwardPort, err)
	}

	// make sure all users can access the socket
	_ = os.Chmod(runnerProxySocket, 0o777)

	// add all listeners to the list
	s.listeners = append(s.listeners, sshListener, wsListener, runnerProxyListener)

	// Setup HTTP handler for git credentials
	go func() {
		mux := http.NewServeMux()
		transport := &http.Transport{DialContext: s.tsServer.Dial}
		mux.HandleFunc("/git-credentials", func(w http.ResponseWriter, r *http.Request) {
			s.gitCredentialsHandler(w, r, lc, transport, projectName, workspaceName)
		})
		if err := http.Serve(runnerProxyListener, mux); err != nil && err != http.ErrServerClosed {
			s.log.Errorf("HTTP runner proxy server error: %v", err)
		}
	}()

	// Setup HTTP handler for port forwarding.
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/portforward", s.httpPortForwardHandler)
		if err := http.Serve(wsListener, mux); err != nil && err != http.ErrServerClosed {
			s.log.Errorf("HTTP server error on TS port %s: %v", TSPortForwardPort, err)
		}
	}()

	// Start handling SSH connections.
	go s.handleSSHConnections(ctx, sshListener)

	return nil
}

// createListener creates a raw listener and wraps it with connection tracking.
func (s *WorkspaceServer) createListener(addr string) (net.Listener, error) {
	l, err := s.tsServer.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// create a new tracked listener to track the number of connections
	return l, nil
}

func (s *WorkspaceServer) addConnection() {
	s.connectionCounterMu.Lock()
	defer s.connectionCounterMu.Unlock()
	s.connectionCounter++
}

func (s *WorkspaceServer) removeConnection() {
	s.connectionCounterMu.Lock()
	defer s.connectionCounterMu.Unlock()
	s.connectionCounter--
}

// httpPortForwardHandler is the HTTP reverse proxy handler for workspace.
// It reconstructs the target URL using custom headers and forwards the request.
func (s *WorkspaceServer) gitCredentialsHandler(w http.ResponseWriter, r *http.Request, lc *tailscale.LocalClient, transport *http.Transport, projectName, workspaceName string) {
	s.log.Infof("Received git credentials request from %s", r.RemoteAddr)

	// create a new http client with a custom transport
	discoveredRunner, err := s.discoverRunner(r.Context(), lc)
	if err != nil {
		http.Error(w, "failed to discover runner", http.StatusInternalServerError)
		return
	}

	// build the runner URL
	runnerURL := fmt.Sprintf("http://%s.ts.loft/devpod/%s/%s/workspace-git-credentials", discoveredRunner, projectName, workspaceName)
	parsedURL, err := url.Parse(runnerURL)
	if err != nil {
		http.Error(w, "failed to parse runner URL", http.StatusInternalServerError)
		return
	}

	// Build the reverse proxy with a custom Director.
	proxy := httputil.NewSingleHostReverseProxy(parsedURL)
	proxy.Director = func(req *http.Request) {
		dest := *parsedURL
		req.URL = &dest
		req.Host = dest.Host
		req.Header.Set("Authorization", "Bearer "+s.config.AccessKey)
	}
	proxy.Transport = transport
	proxy.ServeHTTP(w, r)
}

// httpPortForwardHandler is the HTTP reverse proxy handler for workspace.
// It reconstructs the target URL using custom headers and forwards the request.
func (s *WorkspaceServer) httpPortForwardHandler(w http.ResponseWriter, r *http.Request) {
	s.addConnection()
	defer s.removeConnection()
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
		go s.handleSSHConnection(clientConn)
	}
}

// handleSSHConnection proxies the SSH connection to the local backend.
func (s *WorkspaceServer) handleSSHConnection(clientConn net.Conn) {
	s.addConnection()
	defer s.removeConnection()
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

func (s *WorkspaceServer) sendHeartbeats(ctx context.Context, projectName, workspaceName string, lc *tailscale.LocalClient) {
	// create a new http client with a custom transport
	transport := &http.Transport{DialContext: s.tsServer.Dial}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

	// create a ticker to send heartbeats every 10 seconds
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// get the current number of connections
			s.connectionCounterMu.Lock()
			connections := s.connectionCounter
			s.connectionCounterMu.Unlock()

			// send a heartbeat if there are connections
			if connections > 0 {
				err := s.sendHeartbeat(ctx, client, projectName, workspaceName, lc, connections)
				if err != nil {
					s.log.Errorf("Failed to send heartbeat: %v", err)
				}
			}
		}
	}
}

func (s *WorkspaceServer) sendHeartbeat(ctx context.Context, client *http.Client, projectName, workspaceName string, lc *tailscale.LocalClient, connections int) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	discoveredRunner, err := s.discoverRunner(ctx, lc)
	if err != nil {
		return fmt.Errorf("failed to discover runner: %w", err)
	}

	heartbeatURL := fmt.Sprintf("http://%s.ts.loft/devpod/%s/%s/heartbeat", discoveredRunner, projectName, workspaceName)
	s.log.Infof("Sending heartbeat to %s, because there are %d active connections", heartbeatURL, connections)
	req, err := http.NewRequestWithContext(ctx, "GET", heartbeatURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %w", heartbeatURL, err)
	}

	req.Header.Set("Authorization", "Bearer "+s.config.AccessKey)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s failed: %w", heartbeatURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received response from %s - Status: %d", heartbeatURL, resp.StatusCode)
	}
	s.log.Infof("received response from %s - Status: %d", heartbeatURL, resp.StatusCode)
	return nil
}

// discoverRunner attempts to find the runner peer from the TSNet status.
func (s *WorkspaceServer) discoverRunner(ctx context.Context, lc *tailscale.LocalClient) (string, error) {
	status, err := lc.Status(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	var runner string
	for _, peer := range status.Peer {
		if peer == nil || peer.HostName == "" {
			continue
		}

		if strings.HasSuffix(peer.HostName, "runner") {
			runner = peer.HostName
			break
		}
	}
	if runner == "" {
		return "", fmt.Errorf("no active runner found")
	}

	s.log.Infof("discoverRunner: selected runner = %s", runner)
	return runner, nil
}
