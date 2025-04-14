package local

import (
	"context"
	"fmt"
	"net"
	"time"

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
	LocalCredentialsServerPort int    = 9999
	DefaultTargetHost          string = "localhost"
)

type LocalCredentialsServerProxy struct {
	log        log.Logger
	tsServer   *tsnet.Server
	grpcServer *grpc.Server
	ln         net.Listener
}

func NewLocalCredentialsServerProxy(tsServer *tsnet.Server, logger log.Logger) (*LocalCredentialsServerProxy, error) {
	logger.Infof("NewLocalCredentialsServerProxy: initializing local reverse proxy")
	if tsServer == nil {
		return nil, fmt.Errorf("tsnet.Server cannot be nil")
	}
	return &LocalCredentialsServerProxy{
		log:      logger,
		tsServer: tsServer,
	}, nil
}

func (s *LocalCredentialsServerProxy) Listen(ctx context.Context) error {
	s.log.Infof("LocalCredentialsServerProxy: Starting reverse proxy on tsnet port %d", LocalCredentialsServerPort)

	listenAddr := fmt.Sprintf(":%d", LocalCredentialsServerPort)
	ln, err := s.tsServer.Listen("tcp", listenAddr)
	if err != nil {
		s.log.Errorf("LocalCredentialsServerProxy: Failed to listen on tsnet %s: %v", listenAddr, err)
		return fmt.Errorf("failed to listen on tsnet %s: %w", listenAddr, err)
	}
	s.ln = ln

	s.log.Infof("LocalCredentialsServerProxy: tsnet listener started on %s", ln.Addr().String())

	director := func(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, nil, status.Errorf(codes.InvalidArgument, "missing metadata")
		}

		// Get the target port from metadata. Host is always localhost.
		targetPorts := md.Get("x-target-port")
		if len(targetPorts) == 0 {
			s.log.Error("LocalCredentialsServerProxy: Director missing x-target-port metadata")
			return nil, nil, status.Errorf(codes.InvalidArgument, "missing x-target-port metadata")
		}
		// targetPort := targetPorts[0]
		targetPort := "4795" // FIXME

		targetAddr := net.JoinHostPort(DefaultTargetHost, targetPort)

		s.log.Infof("[LocalCredentialsServerProxy] [gRPC] Proxying call %q to target %s", fullMethodName, targetAddr)

		conn, err := grpc.DialContext(ctx, targetAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithCodec(proxy.Codec()), // Use proxy codec for transparency
		)
		if err != nil {
			s.log.Errorf("[LocalCredentialsServerProxy] [gRPC] Failed to dial local target backend %s: %v", targetAddr, err)
			return nil, nil, status.Errorf(codes.Internal, "failed to dial local target backend: %v", err)
		}

		return ctx, conn, nil
	}

	// Create the gRPC server using the transparent handler.
	// It will forward any unknown service call based on the director logic.
	s.grpcServer = grpc.NewServer(
		grpc.UnknownServiceHandler(proxy.TransparentHandler(director)),
	)

	s.log.Infof("LocalCredentialsServerProxy: gRPC reverse proxy configured, starting server on %s", ln.Addr().String())

	if err := s.grpcServer.Serve(s.ln); err != nil {
		if err.Error() != "grpc: the server has been stopped" {
			s.log.Errorf("LocalCredentialsServerProxy: failed to serve: %v", err)
			return fmt.Errorf("gRPC server error: %w", err)
		} else {
			s.log.Infof("LocalCredentialsServerProxy: gRPC server stopped gracefully.")
		}
	}
	return nil
}

func (s *LocalCredentialsServerProxy) Stop() {
	s.log.Info("LocalCredentialsServerProxy: Stopping reverse proxy...")
	if s.grpcServer != nil {
		stopped := make(chan struct{})
		go func() {
			s.grpcServer.GracefulStop()
			close(stopped)
		}()

		select {
		case <-time.After(10 * time.Second):
			s.log.Warnf("LocalCredentialsServerProxy: Graceful shutdown timed out after 10 seconds, forcing stop.")
			s.grpcServer.Stop()
		case <-stopped:
			s.log.Infof("LocalCredentialsServerProxy: gRPC server stopped gracefully.")
		}
	}

	if s.ln != nil {
		if err := s.ln.Close(); err != nil {
			s.log.Errorf("LocalCredentialsServerProxy: Error closing listener: %v", err)
		} else {
			s.log.Infof("LocalCredentialsServerProxy: Listener closed.")
		}
	}
	s.log.Info("LocalCredentialsServerProxy: Reverse proxy stopped.")
}
