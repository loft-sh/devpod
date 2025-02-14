package ts

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"time"

	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"

	"github.com/loft-sh/devpod/pkg/platform/client"
	sshServer "github.com/loft-sh/devpod/pkg/ssh/server"
	"k8s.io/klog/v2"
	"tailscale.com/envknob"
	"tailscale.com/ipn/store"
	"tailscale.com/tsnet"
)

type WorkspaceServer struct {
	tsServer  *tsnet.Server
	listeners []net.Listener

	config *WorkspaceServerConfig
}

// WorkspaceServerConfig defines the configuration for the TSNet instance
type WorkspaceServerConfig struct {
	AccessKey string
	Host      string
	Hostname  string
	LogF      func(format string, args ...any)
	Client    client.Client
}

// NewTSNet creates a new instance of TSNet
func NewWorkspaceServer(config *WorkspaceServerConfig) *WorkspaceServer {
	return &WorkspaceServer{
		config: config,
	}
}

// Start runs tailscale up and binds port handlers
func (t *WorkspaceServer) Start(ctx context.Context) error {
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
	s, err := store.NewFileStore(t.config.LogF, "/tmp/tailscale/state") // FIXME: proper location
	if err != nil {
		return fmt.Errorf("Create file store: %w", err)
	}
	envknob.Setenv("TS_DEBUG_TLS_DIAL_INSECURE_SKIP_VERIFY", "true")
	klog.Infof("Connecting to control URL - %v", baseUrl.String()+"/coordinator/")
	t.tsServer = &tsnet.Server{
		Hostname:   t.config.Hostname,
		Logf:       t.config.LogF,
		ControlURL: baseUrl.String() + "/coordinator/",
		AuthKey:    t.config.AccessKey,
		Dir:        "/tmp/tailscale/runner", // FIXME: proper location
		Ephemeral:  false,
		Store:      s,
	}

	// Start the server
	if _, err := t.tsServer.Up(ctx); err != nil {
		return fmt.Errorf("failed to start tsnet server: %w", err)
	}

	listener, err := t.tsServer.Listen("tcp", fmt.Sprintf(":%d", sshServer.DefaultUserPort))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", sshServer.DefaultUserPort, err)
	}
	t.listeners = append(t.listeners, listener)

	go func() {
		gracePeriod := 5 * time.Second // TODO: make configurable
		counter := newConnectionCounter(context.TODO(), gracePeriod, func(address string) {
			// TODO:Update sleep mode
		}, log.Default.WithLevel(logrus.DebugLevel))

		for {
			// client conn is the one coming in from tailscale
			// backend conn is our connection to the ssh server
			clientConn, err := listener.Accept()
			if err != nil {
				klog.Errorf("Failed to accept connection: %v", err)
				continue
			}
			clientHost, _, err := net.SplitHostPort(clientConn.RemoteAddr().String())
			if err != nil {
				clientConn.Close()
				fmt.Println("Unable to parse host", clientConn.RemoteAddr().String())
				continue
			}
			counter.Add(clientHost)

			go func(clientConn net.Conn) {
				defer clientConn.Close()
				defer counter.Dec(clientHost)

				localAddr := fmt.Sprintf("127.0.0.1:%d", sshServer.DefaultUserPort)
				backendConn, err := net.Dial("tcp", localAddr)
				if err != nil {
					klog.Errorf("Failed to connect to local address %s: %v", localAddr, err)
					return
				}
				defer backendConn.Close()

				go func() {
					defer clientConn.Close()
					defer backendConn.Close()
					_, err = io.Copy(backendConn, clientConn)
				}()
				_, err = io.Copy(clientConn, backendConn)
			}(clientConn)
		}
	}()

	<-ctx.Done()
	return nil
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

	klog.Info("Tailscale server stopped")
}

// Dial allows dialing to a specific address via Tailscale
func (t *WorkspaceServer) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	if t.tsServer == nil {
		return nil, fmt.Errorf("tailscale server is not running")
	}
	return t.tsServer.Dial(ctx, network, addr)
}
