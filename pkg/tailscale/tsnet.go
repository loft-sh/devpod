package tailscale

import (
	"cmp"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/log"
	"k8s.io/klog/v2"
	"tailscale.com/envknob"
	"tailscale.com/ipn/store/mem"
	"tailscale.com/tsnet"
	"tailscale.com/types/logger"
)

type Connection struct {
	AccessKey     string
	Host          string
	Project       string
	Context       string
	Provider      string
	WorkspaceName string
	CaData        []byte
	Insecure      bool
}

func NewPlatformClient(ctx context.Context, opts *Connection) (client.Client, error) {
	conf, err := config.LoadConfig(opts.Context, opts.Provider)
	if err != nil {
		return nil, err
	}

	loftClient, err := platform.InitClientFromHost(ctx, conf, opts.Host, &log.StreamLogger{})
	if err != nil {
		return nil, err
	}

	return loftClient, nil
}

var ErrNoAccessKeyAndHost = errors.New("accesskey and host empty")

type TSNet interface {
	Start(ctx context.Context, connection *Connection) error
	Stop()
}

type tsNet struct {
	m sync.Mutex

	tsCtx  context.Context
	cancel context.CancelFunc

	proxyServer      *http.Server
	tsServerListener net.Listener
	tsServer         *tsnet.Server

	log klog.Logger
}

func NewTSNet(ctx context.Context) TSNet {
	return &tsNet{
		log: klog.FromContext(ctx).WithName("ts-net-controller"),
	}
}

func (t *tsNet) Start(ctx context.Context, connection *Connection) error {
	t.m.Lock()
	defer t.m.Unlock()

	// we need that context to cleanup the watcher
	t.tsCtx, t.cancel = context.WithCancel(ctx)
	klog.Infof("Start tsnet, stating server")
	// set up the servers
	t.log.Info("starting tsNet server")
	err := t.setupServers(ctx, connection)
	klog.Infof("setup servers successful")

	if err != nil {
		t.log.Error(err, "failed to start tsnet server")
		return err
	}

	// serve the proxy server
	go func() {
		klog.Infof("Serve the proxy server")
		if serveError := t.proxyServer.Serve(t.tsServerListener); serveError != nil && !errors.Is(serveError, http.ErrServerClosed) {
			t.log.Error(serveError, "failed to start tsnet api proxy")
			return
		}
	}()

	// start the watcher
	t.log.V(1).Info("tsNet server started, kicking off watcher and shutdown handler")
	go t.runWatcher(t.tsCtx, connection)
	return nil
}

func (t *tsNet) Stop() {
	t.m.Lock()
	defer t.m.Unlock()

	// check if it was started before
	t.log.V(1).Info("received stop signal")
	if t.tsServer == nil {
		t.log.V(1).Info("tsServer is nil, nothing to do")
		return
	}

	// stop watcher
	t.cancel()

	// close ts net server first
	if err := t.tsServer.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		t.log.Error(err, "Failed to stop tsnet server")
	}

	// close the proxy server next
	if err := t.proxyServer.Close(); err != nil {
		t.log.Error(err, "Failed to stop proxy proxy")
	}

	// close the ts server listener
	if err := t.tsServerListener.Close(); err != nil {
		t.log.Error(err, "Failed to stop tsnet server listener")
	}

	// make sure to get rid of those references
	t.tsCtx = nil
	t.cancel = nil
	t.tsServer = nil
	t.tsServerListener = nil
	t.proxyServer = nil
}

func (t *tsNet) runWatcher(ctx context.Context, connection *Connection) {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	failCounter := 0
	for {
		select {
		case <-t.tsCtx.Done():
			return
		case <-ticker.C:
			// if we are over 12 fails, we restart the server (120 seconds)
			if failCounter > 12 {
				t.log.Info("tsnet server is not running, initiating restart")

				// restart ourselves because it seems the tsnet server is not running
				t.Stop()
				err := t.Start(context.WithoutCancel(ctx), connection)
				if err != nil {
					t.log.Error(err, "cannot start tsNet")
				}
				return
			}

			// check if we are online
			err := t.isTSNetOnline(ctx, connection)
			if err != nil {
				t.log.Error(err, "Check if TSNet is online", "failCounter", failCounter)
				failCounter++
				continue
			}

			// reset fail counter
			failCounter = 0
		}
	}
}

func (t *tsNet) isTSNetOnline(ctx context.Context, connection *Connection) error {
	platformURL := url.URL{
		Scheme: cmp.Or(os.Getenv("LOFT_TSNET_SCHEME"), "https"),
		Host:   connection.Host,
		Path:   "/coordinator/",
	}

	if err := checkDerpConnection(ctx, platformURL); err != nil {
		return fmt.Errorf("check derp connection: %w", err)
	}

	lc, err := t.tsServer.LocalClient()
	if err != nil {
		return fmt.Errorf("get local client: %w", err)
	}

	status, err := lc.Status(ctx)
	if err != nil {
		return fmt.Errorf("get status of local client: %w", err)
	}

	if status.Self == nil {
		return fmt.Errorf("get self status from local client: is nil")
	}

	if !status.Self.Online || !status.Self.InNetworkMap {
		return fmt.Errorf("vcluster tsnet server is not online: %v", status)
	}

	return nil
}

func (t *tsNet) setupServers(ctx context.Context, connection *Connection) error {
	var err error

	// return empty function if there is no access key or host
	if connection.AccessKey == "" || connection.Host == "" {
		return ErrNoAccessKeyAndHost
	}

	// set instance name for connection if its
	if connection.WorkspaceName == "" {
		connection.WorkspaceName = "dummy-workspace-name-2" // FIXME
	}

	// find the platform additional CA
	if len(connection.CaData) > 0 {
		envknob.Setenv("TS_DEBUG_TLS_DIAL_ADDITIONAL_CA_B64", base64.StdEncoding.EncodeToString(connection.CaData))
	}

	// is insecure?
	if connection.Insecure {
		if err := os.Setenv("TS_DEBUG_TLS_DIAL_INSECURE_SKIP_VERIFY", "true"); err != nil {
			return fmt.Errorf("failed to set insecure env var: %w", err)
		}

		envknob.Setenv("TS_DEBUG_TLS_DIAL_INSECURE_SKIP_VERIFY", "true")
	} else {
		envknob.Setenv("TS_DEBUG_TLS_DIAL_INSECURE_SKIP_VERIFY", "false")
	}

	platformURL := url.URL{
		Scheme: cmp.Or(os.Getenv("LOFT_TSNET_SCHEME"), "https"),
		Host:   connection.Host,
		Path:   "/coordinator/",
	}

	// make sure before we do serious tsnet connections that the platform derp is actually reachable
	if err := checkDerpConnection(ctx, platformURL); err != nil {
		return fmt.Errorf("failed to check DERP connection for tsnet: %w", err)
	}

	// start tsnet server
	store, _ := mem.New(logger.Discard, "")
	t.tsServer = &tsnet.Server{
		Dir:        "/tmp/tailscale",
		Store:      store,
		Hostname:   fmt.Sprintf("%s-%s", connection.WorkspaceName, connection.Project),
		Logf:       tsnetLogger(ctx),
		Ephemeral:  true,
		AuthKey:    connection.AccessKey,
		ControlURL: platformURL.String(),
	}

	klog.Infof("OK GOT TS SERVER -> %v", t.tsServer)
	klog.Infof("Starting tsnet.Server with AuthKey: %s and ControlURL: %s", t.tsServer.AuthKey, t.tsServer.ControlURL)

	// we start listening on :80 inside the tailnet (not locally on port :80)
	t.log.V(1).Info("setting up listener")
	t.tsServerListener, err = t.tsServer.Listen("tcp", ":80")
	if err != nil {
		return fmt.Errorf("tsserver listen: %w", err)
	}

	if err != nil {
		return fmt.Errorf("kubeapi proxy server: %w", err)
	}

	return nil
}

// tsnetLogger returns a logger that logs to klog if the LOFT_LOG_TSNET
// environment variable is set to true.
func tsnetLogger(ctx context.Context) logger.Logf {
	logf := func(format string, args ...any) {
		klog.FromContext(ctx).WithName("tailscale").V(1).Info(fmt.Sprintf(format, args...))
	}
	return logf
}

func checkDerpConnection(ctx context.Context, baseURL url.URL) error {
	transport := CloneDefaultTransport()
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: os.Getenv("TS_DEBUG_TLS_DIAL_INSECURE_SKIP_VERIFY") == "true",
	}

	c := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	baseURL.Path = "/derp/probe"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	res, err := c.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		klog.FromContext(ctx).Error(err, "Failed to reach the coordinator server. Make sure that the vCluster can reach the platform. Also, make sure to try using `platform.api.insecure` in the vcluster.yaml in case of x509 certificate issues.")
		return fmt.Errorf("failed to reach the coordinator server: %w", err)
	}

	klog.Infof("Got derp response, response: %v, statusCode: %v", res.Body, res.Status)

	return nil
}

func CloneDefaultTransport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// we disable http2 as Kubernetes has problems with this
	transport.ForceAttemptHTTP2 = false
	return transport
}

func RemoveProtocol(hostPath string) string {
	if idx := strings.Index(hostPath, "://"); idx != -1 {
		return hostPath[idx+3:]
	}
	return hostPath
}
