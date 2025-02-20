package ts

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"

	"github.com/loft-sh/devpod/pkg/platform/client"
	sshServer "github.com/loft-sh/devpod/pkg/ssh/server"
	"tailscale.com/envknob"
	"tailscale.com/ipn/store"
	"tailscale.com/tsnet"
)

type WorkspaceServer struct {
	tsServer  *tsnet.Server
	listeners []net.Listener

	config *WorkspaceServerConfig
	log    log.Logger
}

// WorkspaceServerConfig defines the configuration for the TSNet instance
type WorkspaceServerConfig struct {
	AccessKey string
	Host      string
	Hostname  string
	LogF      func(format string, args ...any)
	Client    client.Client
}

// NewWorkspaceServer creates a new instance of TSNet
func NewWorkspaceServer(config *WorkspaceServerConfig, log log.Logger) *WorkspaceServer {
	return &WorkspaceServer{
		config: config,
		log:    log,
	}
}

// Start runs tailscale up and binds port handlers
func (t *WorkspaceServer) Start(ctx context.Context) error {
	// Validate configuration.
	if err := t.validateConfig(); err != nil {
		return err
	}

	// Setup control URL and check DERP connection.
	baseURL, err := t.setupControlURL(ctx)
	if err != nil {
		return err
	}

	// Initialize Tailscale server.
	if err := t.initTsServer(ctx, baseURL); err != nil {
		return err
	}

	// Create listener.
	listener, err := t.createListener()
	if err != nil {
		return err
	}

	// Parse hostname to extract workspace and project names.
	workspaceName, projectName, err := t.parseHostname()
	if err != nil {
		return err
	}

	// Discover runner from Tailscale peers.
	discoveredRunner := t.discoverRunner(ctx)

	// Create connection counter with heartbeat callbacks.
	counter := t.createConnectionCounter(ctx, discoveredRunner, projectName, workspaceName)

	// Start handling incoming connections.
	t.handleIncomingConnections(ctx, listener, counter)

	// Wait until context is cancelled.
	<-ctx.Done()
	return nil
}

// validateConfig ensures that required configuration values are set.
func (t *WorkspaceServer) validateConfig() error {
	if t.config.AccessKey == "" || t.config.Host == "" {
		return fmt.Errorf("access key or host cannot be empty")
	}
	return nil
}

// setupControlURL constructs the control URL and verifies DERP connection.
func (t *WorkspaceServer) setupControlURL(ctx context.Context) (*url.URL, error) {
	baseURL := &url.URL{
		Scheme: GetEnvOrDefault("LOFT_TSNET_SCHEME", "https"),
		Host:   t.config.Host,
	}
	if err := CheckDerpConnection(ctx, baseURL); err != nil {
		return nil, fmt.Errorf("failed to verify DERP connection: %w", err)
	}
	return baseURL, nil
}

// initTsServer initializes the TSNet server with the provided control URL.
func (t *WorkspaceServer) initTsServer(ctx context.Context, controlURL *url.URL) error {
	s, err := store.NewFileStore(t.config.LogF, "/tmp/tailscale/state") // FIXME: update path as needed
	if err != nil {
		return fmt.Errorf("failed to create file store: %w", err)
	}
	envknob.Setenv("TS_DEBUG_TLS_DIAL_INSECURE_SKIP_VERIFY", "true")
	t.log.Infof("Connecting to control URL - %v", controlURL.String()+"/coordinator/")
	t.tsServer = &tsnet.Server{
		Hostname:   t.config.Hostname,
		Logf:       t.config.LogF,
		ControlURL: controlURL.String() + "/coordinator/",
		AuthKey:    t.config.AccessKey,
		Dir:        "/tmp/tailscale/runner", // FIXME: update path as needed
		Ephemeral:  false,
		Store:      s,
	}

	if _, err := t.tsServer.Up(ctx); err != nil {
		return fmt.Errorf("failed to start tsnet server: %w", err)
	}
	return nil
}

// createListener creates a TCP listener on the default SSH server port.
func (t *WorkspaceServer) createListener() (net.Listener, error) {
	listener, err := t.tsServer.Listen("tcp", fmt.Sprintf(":%d", sshServer.DefaultUserPort))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %d: %w", sshServer.DefaultUserPort, err)
	}
	t.listeners = append(t.listeners, listener)
	return listener, nil
}

// parseHostname extracts the workspace and project names from the hostname.
func (t *WorkspaceServer) parseHostname() (string, string, error) {
	parts := strings.Split(t.config.Hostname, ".")
	if len(parts) < 4 {
		return "", "", fmt.Errorf("invalid hostname format: %s", t.config.Hostname)
	}
	return parts[1], parts[2], nil
}

// discoverRunner performs peer discovery to identify a runner.
func (t *WorkspaceServer) discoverRunner(ctx context.Context) string {
	var discoveredRunner string
	localClient, err := t.tsServer.LocalClient()
	if err != nil {
		t.log.Infof("Failed to get local client: %v", err)
		return discoveredRunner
	}
	status, err := localClient.Status(ctx)
	if err != nil {
		t.log.Infof("Failed to retrieve Tailscale status: %v", err)
		return discoveredRunner
	}
	if status == nil {
		t.log.Infof("Tailscale status is nil")
		return discoveredRunner
	}
	for _, peerStatus := range status.Peer {
		if peerStatus == nil {
			continue
		}
		hostname := peerStatus.HostName
		if hostname == "" {
			continue
		}
		t.log.Infof("Discovered peer %s: Tailscale IPs: %v", hostname, peerStatus.TailscaleIPs)
		if strings.HasSuffix(hostname, "runner") && discoveredRunner == "" {
			discoveredRunner = hostname
		}
	}
	return discoveredRunner
}

// createConnectionCounter sets up the connection counter with heartbeat callbacks.
func (t *WorkspaceServer) createConnectionCounter(ctx context.Context, discoveredRunner, projectName, workspaceName string) *connectionCounter {
	var (
		hbMu       sync.Mutex
		heartbeats = make(map[string]context.CancelFunc)
	)

	onConnect := func(address string) {
		if discoveredRunner == "" {
			t.log.Infof("client %s connected, but no runner discovered", address)
			return
		}
		t.log.Infof("onConnect: client %s connected", address)

		hbMu.Lock()
		if _, exists := heartbeats[address]; exists {
			hbMu.Unlock()
			return
		}
		hbCtx, cancel := context.WithCancel(ctx)
		heartbeats[address] = cancel
		hbMu.Unlock()

		heartbeatURL := fmt.Sprintf("http://%s.ts.loft/devpod/%s/%s/heartbeat", discoveredRunner, projectName, workspaceName)
		transport := &http.Transport{DialContext: t.tsServer.Dial}
		client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

		go func(addr string, ctxHB context.Context) {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctxHB.Done():
					t.log.Infof("Heartbeat for client %s stopped", address)
					return
				case <-ticker.C:
					req, err := http.NewRequestWithContext(ctxHB, "GET", heartbeatURL, nil)
					if err != nil {
						t.log.Infof("Heartbeat: failed to create request for %s: %v", heartbeatURL, err)
						continue
					}
					req.Header.Set("Authorization", "Bearer "+t.config.AccessKey)
					resp, err := client.Do(req)
					if err != nil {
						t.log.Infof("Heartbeat: request to %s failed: %v", heartbeatURL, err)
						continue
					}
					t.log.Infof("Heartbeat: received response from %s - Status: %d", heartbeatURL, resp.StatusCode)
				}
			}
		}(address, hbCtx)
	}

	onDisconnect := func(address string) {
		t.log.Infof("client %s has no active connections, stopping heartbeat", address)
		hbMu.Lock()
		if cancel, exists := heartbeats[address]; exists {
			cancel()
			delete(heartbeats, address)
		}
		hbMu.Unlock()
	}

	gracePeriod := 5 * time.Second
	return newConnectionCounter(ctx, gracePeriod, onConnect, onDisconnect, log.Default.WithLevel(logrus.DebugLevel))
}

// handleIncomingConnections accepts incoming connections and proxies them to the local SSH server.
func (t *WorkspaceServer) handleIncomingConnections(ctx context.Context, listener net.Listener, counter *connectionCounter) {
	go func() {
		for {
			clientConn, err := listener.Accept()
			if err != nil {
				t.log.Errorf("Failed to accept connection: %v", err)
				continue
			}
			clientHost, _, err := net.SplitHostPort(clientConn.RemoteAddr().String())
			if err != nil {
				clientConn.Close()
				t.log.Infof("Unable to parse host: %v", clientConn.RemoteAddr().String())
				continue
			}
			counter.Add(clientHost)

			go func(clientConn net.Conn, clientHost string) {
				defer clientConn.Close()
				defer counter.Dec(clientHost)

				localAddr := fmt.Sprintf("127.0.0.1:%d", sshServer.DefaultUserPort)
				backendConn, err := net.Dial("tcp", localAddr)
				if err != nil {
					t.log.Errorf("Failed to connect to local address %s: %v", localAddr, err)
					return
				}
				defer backendConn.Close()

				go func() {
					defer clientConn.Close()
					defer backendConn.Close()
					_, err = io.Copy(backendConn, clientConn)
				}()
				_, err = io.Copy(clientConn, backendConn)
			}(clientConn, clientHost)
		}
	}()
}

// Stop stops the TSNet server and closes all listeners
func (t *WorkspaceServer) Stop() {
	for _, listener := range t.listeners {
		if listener != nil {
			listener.Close()
		}
	}

	if t.tsServer != nil {
		t.tsServer.Close()
		t.tsServer = nil
	}

	t.log.Info("Tailscale server stopped")
}

// Dial allows dialing to a specific address via Tailscale
func (t *WorkspaceServer) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	if t.tsServer == nil {
		return nil, fmt.Errorf("tailscale server is not running")
	}
	return t.tsServer.Dial(ctx, network, addr)
}
