package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/loft-sh/log"
	"github.com/mwitkow/grpc-proxy/proxy"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"tailscale.com/tsnet"
)

const (
	HeaderTargetHost string = "x-target-host"
	HeaderTargetPort string = "x-target-port"
	HeaderProxyPort  string = "x-proxy-port"
)

// NetworkProxyService proxies gRPC and HTTP requests over DevPod network.
// It coordinates the listener, cmux, and underlying servers.
type NetworkProxyService struct {
	mainListener net.Listener
	grpcServer   *grpc.Server
	httpServer   *http.Server
	tsServer     *tsnet.Server
	log          log.Logger
	socketPath   string
	mux          cmux.CMux
	grpcDirector *GrpcDirector
	httpProxy    *HttpProxyHandler
}

// NewNetworkProxyService creates a new instance listening on the given unix socket.
func NewNetworkProxyService(socketPath string, tsServer *tsnet.Server, log log.Logger) (*NetworkProxyService, error) {
	_ = os.Remove(socketPath)
	l, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on socket %s: %w", socketPath, err)
	}

	if err := os.Chmod(socketPath, 0777); err != nil {
		l.Close()
		return nil, fmt.Errorf("failed to set socket permissions on %s: %w", socketPath, err)
	}

	log.Infof("NetworkProxyService: network proxy listening on socket %s", socketPath)

	grpcDirector := NewGrpcDirector(tsServer, log)
	httpProxy := NewHttpProxyHandler(tsServer, log)

	return &NetworkProxyService{
		mainListener: l,
		tsServer:     tsServer,
		log:          log,
		socketPath:   socketPath,
		grpcDirector: grpcDirector,
		httpProxy:    httpProxy,
	}, nil
}

// Start runs the gRPC reverse proxy server.
func (s *NetworkProxyService) Start(ctx context.Context) error {
	// Create connection multiplexer
	s.mux = cmux.New(s.mainListener)

	// Matchers
	grpcL := s.mux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpL := s.mux.Match(cmux.Any())

	// Servers
	s.grpcServer = grpc.NewServer(
		grpc.UnknownServiceHandler(proxy.TransparentHandler(s.grpcDirector.DirectorFunc)),
	)
	s.httpServer = &http.Server{
		Handler: s.httpProxy,
	}

	// Start servers
	var runWg sync.WaitGroup
	errChan := make(chan error, 3)

	runWg.Add(1)
	go func() {
		defer runWg.Done()
		s.log.Debugf("NetworkProxyService: starting gRPC server...")
		if err := s.grpcServer.Serve(grpcL); err != nil && !errors.Is(err, grpc.ErrServerStopped) && !errors.Is(err, cmux.ErrListenerClosed) {
			s.log.Errorf("NetworkProxyService: gRPC server error: %v", err)
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		} else {
			s.log.Debugf("NetworkProxyService: gRPC server stopped.")
		}
	}()

	runWg.Add(1)
	go func() {
		defer runWg.Done()
		s.log.Debugf("NetworkProxyService: starting HTTP server...")
		if err := s.httpServer.Serve(httpL); err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, cmux.ErrListenerClosed) {
			s.log.Errorf("NetworkProxyService: HTTP server error: %v", err)
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		} else {
			s.log.Debugf("NetworkProxyService: HTTP server stopped.")
		}
	}()

	runWg.Add(1)
	go func() {
		defer runWg.Done()
		s.log.Infof("NetworkProxyService: starting server...")
		err := s.mux.Serve()
		if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, cmux.ErrListenerClosed) {
			s.log.Errorf("NetworkProxyService: server error: %v", err)
			errChan <- fmt.Errorf("server error: %w", err)
		} else {
			s.log.Infof("NetworkProxyService: server stopped.")
		}
	}()

	s.log.Infof("NetworkProxyService: successfully started listeners on %s", s.socketPath)

	var finalErr error
	select {
	case <-ctx.Done():
		s.log.Infof("NetworkProxyService: context cancelled, shutting down proxy service")
		finalErr = ctx.Err()
	case err := <-errChan:
		s.log.Errorf("NetworkProxyService: server error triggered shutdown: %v", err)
		finalErr = err
	}

	s.Stop()

	s.log.Debugf("NetworkProxyService: Waiting for servers to exit...")
	runWg.Wait()
	s.log.Debugf("NetworkProxyService: All servers exited.")

	return finalErr
}

func (s *NetworkProxyService) Stop() {
	s.log.Infof("NetworkProxyService: stopping proxy service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var shutdownWg sync.WaitGroup
	shutdownWg.Add(2)

	go func() {
		defer shutdownWg.Done()
		if s.grpcServer != nil {
			s.grpcServer.GracefulStop()
			s.log.Debugf("NetworkProxyService: gRPC server stopped.")
		}
	}()

	go func() {
		defer shutdownWg.Done()
		if s.httpServer != nil {
			if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
				s.log.Warnf("NetworkProxyService: HTTP server shutdown error: %v", err)
			} else {
				s.log.Debugf("NetworkProxyService: HTTP server stopped.")
			}
		}
	}()

	s.log.Infof("NetworkProxyService: waiting for servers to stop...")

	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		shutdownWg.Wait()
	}()

	select {
	case <-waitDone:
		s.log.Debugf("NetworkProxyService: All server shutdowns completed.")
	case <-shutdownCtx.Done():
		s.log.Warnf("NetworkProxyService: Graceful shutdown timed out after waiting for servers.")
	}

	s.log.Debugf("NetworkProxyService: Listener and socket cleanup.")

	if s.mainListener != nil {
		s.log.Debugf("NetworkProxyService: Closing main listener...")
		if err := s.mainListener.Close(); err != nil {
			if !errors.Is(err, net.ErrClosed) && !errors.Is(err, cmux.ErrListenerClosed) {
				s.log.Errorf("NetworkProxyService: Error closing main listener: %v", err)
			} else {
				s.log.Debugf("NetworkProxyService: Main listener closed.")
			}
		} else {
			s.log.Debugf("NetworkProxyService: Main listener closed successfully.")
		}
	} else {
		s.log.Warnf("NetworkProxyService: Main listener was nil during stop.")
	}

	s.log.Debugf("NetworkProxyService: Removing socket file %s", s.socketPath)
	if err := os.Remove(s.socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		s.log.Warnf("NetworkProxyService: Failed to remove socket file %s: %v", s.socketPath, err)
	} else if err == nil {
		s.log.Debugf("NetworkProxyService: Removed socket file %s", s.socketPath)
	}

	s.log.Infof("NetworkProxyService: Proxy service stopped.")
}
