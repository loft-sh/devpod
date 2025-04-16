package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	"github.com/mwitkow/grpc-proxy/proxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"tailscale.com/tsnet"
)

// NetworkProxyService proxies gRPC requests based on metadata.
type NetworkProxyService struct {
	listener   net.Listener
	grpcServer *grpc.Server
	tsServer   *tsnet.Server
	log        log.Logger
	socketPath string
}

// NewNetworkProxyService creates a new instance listening on the given unix socket.
func NewNetworkProxyService(socketPath string, tsServer *tsnet.Server, log log.Logger) (*NetworkProxyService, error) {
	_ = os.Remove(socketPath)
	l, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on socket %s: %w", socketPath, err)
	}

	if err := os.Chmod(socketPath, 0770); err != nil {
		l.Close()
		return nil, fmt.Errorf("failed to set socket permissions on %s: %w", socketPath, err)
	}

	log.Infof("NetworkProxyService: network proxy listening on socket %s", socketPath)
	return &NetworkProxyService{
		listener:   l,
		tsServer:   tsServer,
		log:        log,
		socketPath: socketPath,
	}, nil
}

// Start runs the gRPC reverse proxy server.
func (s *NetworkProxyService) Start(ctx context.Context) error {
	director := func(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			s.log.Warnf("[NetworkProxyService] [gRPC] Director missing incoming metadata for call %q", fullMethodName)
			return nil, nil, status.Errorf(codes.InvalidArgument, "missing metadata")
		}
		mdCopy := md.Copy()

		targetHosts := mdCopy.Get("x-target-host")
		targetPorts := mdCopy.Get("x-target-port")
		proxyPorts := mdCopy.Get("x-proxy-port")
		if len(targetHosts) == 0 || len(targetPorts) == 0 || len(proxyPorts) == 0 {
			s.log.Errorf("[NetworkProxyService] [gRPC] Director missing x-target-host, x-proxy-port or x-target-port metadata for call %q", fullMethodName)
			return nil, nil, status.Errorf(codes.InvalidArgument, "missing x-target-host, x-proxy-port or x-target-port metadata")
		}

		proxyPort, err := strconv.Atoi(proxyPorts[0])
		if err != nil {
			return nil, nil, err
		}
		targetAddr := ts.EnsureURL(targetHosts[0], proxyPort)

		s.log.Infof("[NetworkProxyService] [gRPC] Proxying call %q to target %s", fullMethodName, targetAddr)

		// Create a custom dialer using the tsnet server.
		tsDialer := func(ctx context.Context, addr string) (net.Conn, error) {
			s.log.Debugf("[NetworkProxyService] [gRPC] Dialing target %s via tsnet", addr)
			conn, err := s.tsServer.Dial(ctx, "tcp", addr)
			if err != nil {
				s.log.Errorf("[NetworkProxyService] [gRPC] Failed to dial target %s via tsnet: %v", addr, err)
				return nil, err
			}
			return conn, nil
		}

		// Dial the target gRPC server (the second proxy) using the tsnet dialer.
		conn, err := grpc.DialContext(ctx, targetAddr,
			grpc.WithContextDialer(tsDialer),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithCodec(proxy.Codec()),
		)
		if err != nil {
			s.log.Errorf("[NetworkProxyService] [gRPC] Failed to dial target backend %s: %v", targetAddr, err)
			return nil, nil, status.Errorf(codes.Internal, "failed to dial target backend: %v", err)
		}

		outCtx := metadata.NewOutgoingContext(ctx, mdCopy)

		return outCtx, conn, nil
	}

	// Create the gRPC server with the transparent proxy handler.
	s.grpcServer = grpc.NewServer(
		grpc.UnknownServiceHandler(proxy.TransparentHandler(director)),
	)

	s.log.Infof("NetworkProxyService: starting gRPC server on %s", s.socketPath)
	go func() {
		if err := s.grpcServer.Serve(s.listener); err != nil && !errors.Is(err, net.ErrClosed) {
			s.log.Errorf("NetworkProxyService: gRPC server error: %v", err)
		} else if errors.Is(err, net.ErrClosed) {
			s.log.Infof("NetworkProxyService: gRPC server stopped gracefully.")
		}
	}()

	<-ctx.Done()
	s.log.Infof("NetworkProxyService: context cancelled, shutting down proxy service")
	s.Stop()
	return ctx.Err()
}

// Stop gracefully shuts down the gRPC server and closes the listener.
func (s *NetworkProxyService) Stop() {
	s.log.Infof("NetworkProxyService: stopping proxy service...")
	stopped := make(chan struct{})
	go func() {
		if s.grpcServer != nil {
			s.grpcServer.GracefulStop()
		}
		close(stopped)
	}()

	// Wait for graceful stop with a timeout
	select {
	case <-time.After(10 * time.Second):
		s.log.Warnf("NetworkProxyService: Graceful stop timed out, forcing listener close.")
	case <-stopped:
		s.log.Infof("NetworkProxyService: gRPC server stopped.")
	}

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				s.log.Errorf("NetworkProxyService: error closing listener: %v", err)
			}
		} else {
			s.log.Infof("NetworkProxyService: Listener closed.")
		}
	}

	if s.socketPath != "" {
		if err := os.Remove(s.socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			s.log.Warnf("NetworkProxyService: failed to remove socket file %s: %v", s.socketPath, err)
		}
	}
	s.log.Infof("NetworkProxyService: proxy service stopped.")
}
