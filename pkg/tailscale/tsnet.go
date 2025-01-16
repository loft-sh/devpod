package tailscale

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"k8s.io/klog/v2"
	"tailscale.com/envknob"
	"tailscale.com/ipn/store/mem"
	"tailscale.com/tsnet"
	tslogger "tailscale.com/types/logger"
)

// TSNet is the main interface
type TSNet interface {
	Start(ctx context.Context) error
	Stop()
	Dial(ctx context.Context, network, addr string) (net.Conn, error)
}

// tsNet is the implementation of TSNet
type tsNet struct {
	tsServer  *tsnet.Server
	listeners []net.Listener
	config    *TSNetConfig
}

// TSNetConfig defines the configuration for the TSNet instance
type TSNetConfig struct {
	AccessKey    string
	Host         string
	Hostname     string
	PortHandlers map[string]func(net.Listener)
	LogF         func(format string, args ...any)
}

// NewTSNet creates a new instance of TSNet
func NewTSNet(config *TSNetConfig) TSNet {
	return &tsNet{
		config: config,
	}
}

// Start starts the TSNet server and binds port handlers
func (t *tsNet) Start(ctx context.Context) error {
	if t.config.AccessKey == "" || t.config.Host == "" {
		return fmt.Errorf("access key or host cannot be empty")
	}

	// Build the platform URL
	baseUrl := url.URL{
		Scheme: getEnvOrDefault("LOFT_TSNET_SCHEME", "https"),
		Host:   t.config.Host,
	}

	// Check DERP connection
	if err := checkDerpConnection(ctx, &baseUrl); err != nil {
		return fmt.Errorf("failed to verify DERP connection: %w", err)
	}

	// Configure the Tailscale server
	store, _ := mem.New(tslogger.Discard, "")
	envknob.Setenv("TS_DEBUG_TLS_DIAL_INSECURE_SKIP_VERIFY", "true")
	klog.Infof("Connecting to control URL - %v", baseUrl.String()+"/coordinator/")
	t.tsServer = &tsnet.Server{
		Hostname:   t.config.Hostname,
		Logf:       t.config.LogF,
		ControlURL: baseUrl.String() + "/coordinator/",
		AuthKey:    t.config.AccessKey,
		Dir:        "/tmp/tailscale/runner",
		Ephemeral:  true,
		Store:      store,
	}

	// Start the server
	if err := t.tsServer.Start(); err != nil {
		return fmt.Errorf("failed to start tsnet server: %w", err)
	}

	// Bind port handlers
	for port, handler := range t.config.PortHandlers {
		listener, err := t.tsServer.Listen("tcp", ":"+port)
		if err != nil {
			return fmt.Errorf("failed to listen on port %s: %w", port, err)
		}
		t.listeners = append(t.listeners, listener)

		go handler(listener)
		klog.Infof("Port %s bound with handler", port)
	}

	<-ctx.Done()
	return nil
}

// Stop stops the TSNet server and closes all listeners
func (t *tsNet) Stop() {
	for _, listener := range t.listeners {
		if listener != nil {
			listener.Close()
		}
	}

	if t.tsServer != nil {
		t.tsServer.Close()
		t.tsServer = nil
	}

	klog.Info("Tailscale server stopped")
}

// Dial allows dialing to a specific address via Tailscale
func (t *tsNet) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	if t.tsServer == nil {
		return nil, fmt.Errorf("Tailscale server is not running")
	}
	return t.tsServer.Dial(ctx, network, addr)
}

// checkDerpConnection validates the DERP connection
func checkDerpConnection(ctx context.Context, baseUrl *url.URL) error {
	newTransport := http.DefaultTransport.(*http.Transport).Clone()
	newTransport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	client := &http.Client{
		Transport: newTransport,
		Timeout:   5 * time.Second,
	}

	derpUrl := *baseUrl
	derpUrl.Path = "/derp/probe"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, derpUrl.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	res, err := client.Do(req)
	if err != nil || (res != nil && res.StatusCode != http.StatusOK) {
		klog.FromContext(ctx).Error(err, "Failed to reach the coordinator server.", "url", derpUrl.String())

		if res != nil {
			body, _ := io.ReadAll(res.Body)
			defer res.Body.Close()
			klog.FromContext(ctx).V(1).Info("Response body", "body", string(body))
		}

		return fmt.Errorf("failed to reach the coordinator server: %w", err)
	}

	return nil
}

// Utility function to get environment variable or default
func getEnvOrDefault(envVar, defaultVal string) string {
	if val := os.Getenv(envVar); val != "" {
		return val
	}
	return defaultVal
}

// ReverseProxyHandler implements TCP reverse proxy
func ReverseProxyHandler(localAddr string) func(net.Listener) {
	return func(listener net.Listener) {
		for {
			clientConn, err := listener.Accept()
			if err != nil {
				klog.Errorf("Failed to accept connection: %v", err)
				continue
			}

			go func(clientConn net.Conn) {
				defer clientConn.Close()

				backendConn, err := net.Dial("tcp", localAddr)
				if err != nil {
					klog.Errorf("Failed to connect to local address %s: %v", localAddr, err)
					return
				}
				defer backendConn.Close()

				go io.Copy(backendConn, clientConn)
				io.Copy(clientConn, backendConn)
			}(clientConn)
		}
	}
}

// RemoveProtocol removes protocol from URL
func RemoveProtocol(hostPath string) string {
	if idx := strings.Index(hostPath, "://"); idx != -1 {
		return hostPath[idx+3:]
	}
	return hostPath
}
