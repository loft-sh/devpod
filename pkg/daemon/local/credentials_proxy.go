package local

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/loft-sh/log"
	"tailscale.com/tsnet"
)

const (
	// Listen on this port via tsnet.
	LocalCredentialsServerPort = 9999 // FIXME - use random prot
	// Target server: local gRPC server running on port 5555.
	TargetServer = "http://localhost:5555" // FIXME - get port from request
)

// LocalCredentialsServerProxy acts as a reverse proxy that blindly forwards
// all incoming traffic to the local gRPC server on port 5555.
type LocalCredentialsServerProxy struct {
	log      log.Logger
	tsServer *tsnet.Server

	ln  net.Listener
	srv *http.Server
}

// NewLocalCredentialsServerProxy initializes a new LocalCredentialsServerProxy.
func NewLocalCredentialsServerProxy(tsServer *tsnet.Server, log log.Logger) (*LocalCredentialsServerProxy, error) {
	return &LocalCredentialsServerProxy{
		log:      log,
		tsServer: tsServer,
	}, nil
}

// Listen creates the tsnet listener and HTTP server,
// and registers a catch-all handler that acts as the reverse proxy.
func (s *LocalCredentialsServerProxy) Listen(ctx context.Context) error {
	s.log.Info("Starting reverse proxy for local gRPC server")

	// Create a tsnet listener.
	ln, err := s.tsServer.Listen("tcp", fmt.Sprintf(":%d", LocalCredentialsServerPort))
	if err != nil {
		s.log.Infof("Failed to listen on tsnet port %d: %v", LocalCredentialsServerPort, err)
		return fmt.Errorf("failed to listen on tsnet port %d: %w", LocalCredentialsServerPort, err)
	}
	s.ln = ln

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleReverseProxy)

	// Create the HTTP server.
	s.srv = &http.Server{
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		s.log.Info("Context canceled, shutting down reverse proxy")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			s.log.Errorf("Error shutting down reverse proxy: %v", err)
		}
	}()

	s.log.Infof("Reverse proxy listening on tsnet port %d", LocalCredentialsServerPort)
	err = s.srv.Serve(ln)
	if err != nil && err != http.ErrServerClosed {
		s.log.Errorf("Reverse proxy error: %v", err)
		return err
	}

	return nil
}

// handleReverseProxy forwards every request to the target gRPC server.
func (s *LocalCredentialsServerProxy) handleReverseProxy(w http.ResponseWriter, r *http.Request) {
	s.log.Infof("Forwarding request %s %s to target server", r.Method, r.URL.String())

	// Parse the target URL.
	targetURL, err := url.Parse(TargetServer)
	if err != nil {
		s.log.Errorf("Error parsing target URL %s: %v", TargetServer, err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// Create the reverse proxy.
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Customize the director to forward the Host header to the target.
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetURL.Host
	}

	// Use an error handler to log any errors that occur.
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		s.log.Errorf("Reverse proxy error: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	// Forward the request.
	proxy.ServeHTTP(w, r)
}

// Close gracefully shuts down the reverse proxy.
func (s *LocalCredentialsServerProxy) Close() error {
	s.log.Info("Closing reverse proxy")
	if s.srv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			s.log.Errorf("Error during reverse proxy shutdown: %v", err)
			return err
		}
	}
	return nil
}
