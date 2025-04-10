package network

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/loft-sh/log"
	"tailscale.com/tsnet"
)

const (
	// TSPortForwardPort is the fixed port on which the workspace HTTP reverse proxy listens.
	TSPortForwardPort string = "12051"

	RunnerProxySocket  string = "runner-proxy.sock"
	NetworkProxySocket string = "devpod-net.sock"
	RootDir            string = "/var/devpod"

	netMapCooldown = 30 * time.Second
)

// HTTPPortForwardService handles HTTP reverse proxy requests.
type HTTPPortForwardService struct {
	listener net.Listener
	tsServer *tsnet.Server
	log      log.Logger
	tracker  *ConnTracker
}

// NewHTTPPortForwardService creates a new HTTPPortForwardService.
func NewHTTPPortForwardService(tsServer *tsnet.Server, tracker *ConnTracker, log log.Logger) (*HTTPPortForwardService, error) {
	l, err := tsServer.Listen("tcp", fmt.Sprintf(":%s", TSPortForwardPort))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on TS port %s: %w", TSPortForwardPort, err)
	}
	return &HTTPPortForwardService{
		listener: l,
		tsServer: tsServer,
		log:      log,
		tracker:  tracker,
	}, nil
}

// Start begins serving HTTP port forwarding requests.
func (s *HTTPPortForwardService) Start(ctx context.Context) {
	s.log.Infof("Starting HTTP reverse proxy listener on TSNet port %s", TSPortForwardPort)
	mux := http.NewServeMux()
	mux.HandleFunc("/portforward", s.portForwardHandler)
	go func() {
		if err := http.Serve(s.listener, mux); err != nil && err != http.ErrServerClosed {
			s.log.Errorf("HTTPPortForwardService error: %v", err)
		}
	}()
}

func (s *HTTPPortForwardService) portForwardHandler(w http.ResponseWriter, r *http.Request) {
	s.tracker.Add()
	defer s.tracker.Remove()
	s.log.Debugf("HTTPPortForwardService: received request")

	targetPort := r.Header.Get("X-Loft-Forward-Port")
	baseForwardStr := r.Header.Get("X-Loft-Forward-Url")
	if targetPort == "" || baseForwardStr == "" {
		http.Error(w, "missing required X-Loft headers", http.StatusBadRequest)
		return
	}
	s.log.Debugf("HTTPPortForwardService: headers: X-Loft-Forward-Port=%s, X-Loft-Forward-Url=%s", targetPort, baseForwardStr)
	parsedURL, err := url.Parse(baseForwardStr)
	if err != nil {
		s.log.Errorf("HTTPPortForwardService: invalid base forward URL: %v", err)
		http.Error(w, "invalid base forward URL", http.StatusBadRequest)
		return
	}
	parsedURL.Scheme = "http"
	parsedURL.Host = "127.0.0.1:" + targetPort
	s.log.Debugf("HTTPPortForwardService: final target URL=%s", parsedURL.String())
	proxy := newReverseProxy(parsedURL, func(h http.Header) {
		h.Del("X-Loft-Forward-Port")
		h.Del("X-Loft-Forward-Url")
		h.Del("X-Loft-Forward-Authorization")
	})
	proxy.Transport = http.DefaultTransport
	s.log.Infof("HTTPPortForwardService: proxying request: %s %s", r.Method, parsedURL.String())
	proxy.ServeHTTP(w, r)
}

// Stop stops the HTTPPortForwardService by closing its listener.
func (s *HTTPPortForwardService) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
}
