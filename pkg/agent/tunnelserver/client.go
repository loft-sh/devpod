package tunnelserver

import (
	"context"
	"io"
	"net"

	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/stdio"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
