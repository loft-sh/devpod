package tunnelserver

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/daemon/workspace/network"
	"github.com/loft-sh/devpod/pkg/stdio"
	"github.com/loft-sh/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
)

func NewTunnelClient(reader io.Reader, writer io.WriteCloser, exitOnClose bool, exitCode int) (tunnel.TunnelClient, error) {
	pipe := stdio.NewStdioStream(reader, writer, exitOnClose, exitCode)

	// After moving from deprecated grpc.Dial to grpc.NewClient we need to setup resolver first
	// https://github.com/grpc/grpc-go/issues/1786#issuecomment-2119088770
	resolver.SetDefaultScheme("passthrough")

	// Set up a connection to the server.
	conn, err := grpc.NewClient("",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return pipe, nil
		}),
	)
	if err != nil {
		return nil, err
	}

	c := tunnel.NewTunnelClient(conn)

	return c, nil
}

// NewHTTPTunnelClient creates a new gRPC client that connects via the network proxy.
func NewHTTPTunnelClient(targetHost string, targetPort string, log log.Logger) (tunnel.TunnelClient, error) {
	resolver.SetDefaultScheme("passthrough")
	log.Infof("Starting tunnel client targeting %s:%s via proxy", targetHost, targetPort)

	// Create a unary interceptor to attach the target metadata.
	unaryInterceptor := func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		md := metadata.New(map[string]string{
			"x-target-host": targetHost,
			"x-target-port": targetPort,
		})
		// Create a new outgoing context with the metadata attached.
		ctx = metadata.NewOutgoingContext(ctx, md)
		log.Debugf("Unary interceptor adding metadata: host=%s, port=%s", targetHost, targetPort)
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	streamInterceptor := func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		md := metadata.New(map[string]string{
			"x-target-host": targetHost,
			"x-target-port": targetPort,
		})
		// Create a new outgoing context with the metadata attached.
		ctx = metadata.NewOutgoingContext(ctx, md)
		log.Debugf("Stream interceptor adding metadata: host=%s, port=%s", targetHost, targetPort)
		return streamer(ctx, desc, cc, method, opts...)
	}

	target := "passthrough:///proxy-socket-target"

	conn, err := grpc.NewClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()), // Connect to proxy socket without TLS
		grpc.WithContextDialer(network.GetContextDialer()),       // Use our custom dialer
		grpc.WithUnaryInterceptor(unaryInterceptor),              // Add metadata for unary calls
		grpc.WithStreamInterceptor(streamInterceptor),            // Add metadata for streaming calls
	)
	if err != nil {
		log.Errorf("Failed to create gRPC client connection via proxy: %v", err)
		return nil, fmt.Errorf("failed to create gRPC client via proxy: %w", err)
	}

	log.Infof("Successfully connected tunnel client via proxy socket")
	c := tunnel.NewTunnelClient(conn)
	return c, nil
}
