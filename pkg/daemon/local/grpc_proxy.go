package local

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/loft-sh/devpod/pkg/daemon/workspace/network"
	"github.com/loft-sh/log"
	"github.com/mwitkow/grpc-proxy/proxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"tailscale.com/tsnet"
)

const (
	DefaultGRPCProxyPort int    = 14798
	DefaultTargetHost    string = "localhost"
)

type LocalGRPCProxy struct {
	log        log.Logger
	tsServer   *tsnet.Server
	grpcServer *grpc.Server
	ln         net.Listener
}

func NewLocalGRPCProxy(tsServer *tsnet.Server, logger log.Logger) (*LocalGRPCProxy, error) {
	if tsServer == nil {
		return nil, fmt.Errorf("tsnet.Server cannot be nil")
	}
	return &LocalGRPCProxy{
		log:      logger,
		tsServer: tsServer,
	}, nil
}

func (s *LocalGRPCProxy) Listen(ctx context.Context) error {
	s.log.Infof("LocalGRPCProxy: Starting reverse proxy on tsnet port %d", DefaultGRPCProxyPort)

	listenAddr := fmt.Sprintf(":%d", DefaultGRPCProxyPort)
	ln, err := s.tsServer.Listen("tcp", listenAddr)
	if err != nil {
		s.log.Errorf("LocalGRPCProxy: Failed to listen on tsnet %s: %v", listenAddr, err)
		return fmt.Errorf("failed to listen on tsnet %s: %w", listenAddr, err)
	}
	s.ln = ln

	s.log.Infof("LocalGRPCProxy: tsnet listener started on %s", ln.Addr().String())

	director := func(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, nil, status.Errorf(codes.InvalidArgument, "missing metadata")
		}

		// Get the target port from metadata. Host is always localhost.
		targetPorts := md.Get(network.HeaderTargetPort)
		if len(targetPorts) == 0 {
			s.log.Error("LocalGRPCProxy: Director missing x-target-port metadata")
			return nil, nil, status.Errorf(codes.InvalidArgument, "missing x-target-port metadata")
		}
		targetPort := targetPorts[0]
		targetAddr := net.JoinHostPort(DefaultTargetHost, targetPort)

		s.log.Infof("LocalGRPCProxy: Proxying call %q to target %s", fullMethodName, targetAddr)

		conn, err := grpc.DialContext(ctx, targetAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithCodec(proxy.Codec()),
		)
		if err != nil {
			s.log.Errorf("LocalGRPCProxy: Failed to dial local target backend %s: %v", targetAddr, err)
			return nil, nil, status.Errorf(codes.Internal, "failed to dial local target backend: %v", err)
		}

		return ctx, conn, nil
	}

	s.grpcServer = grpc.NewServer(
		grpc.UnknownServiceHandler(proxy.TransparentHandler(director)),
	)

	s.log.Debugf("LocalGRPCProxy: gRPC reverse proxy configured, starting server on %s", ln.Addr().String())

	if err := s.grpcServer.Serve(s.ln); err != nil {
		if err.Error() != "grpc: the server has been stopped" {
			s.log.Errorf("LocalGRPCProxy: failed to serve: %v", err)
			return fmt.Errorf("gRPC server error: %w", err)
		} else {
			s.log.Infof("LocalGRPCProxy: server stopped.")
		}
	}
	return nil
}

func (s *LocalGRPCProxy) Stop() {
	s.log.Info("LocalGRPCProxy: Stopping reverse proxy...")
	if s.grpcServer != nil {
		stopped := make(chan struct{})
		go func() {
			s.grpcServer.GracefulStop()
			close(stopped)
		}()

		select {
		case <-time.After(10 * time.Second):
			s.log.Warnf("LocalGRPCProxy: Shutdown timed out after 10 seconds, forcing stop.")
			s.grpcServer.Stop()
		case <-stopped:
			s.log.Infof("LocalGRPCProxy: server stopped.")
		}
	}

	if s.ln != nil {
		if err := s.ln.Close(); err != nil {
			s.log.Errorf("LocalGRPCProxy: Error closing listener: %v", err)
		} else {
			s.log.Infof("LocalGRPCProxy: Listener closed.")
		}
	}
	s.log.Info("LocalGRPCProxy: Reverse proxy stopped.")
}
