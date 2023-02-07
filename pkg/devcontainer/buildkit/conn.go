package buildkit

import (
	"context"
	dockerclient "github.com/docker/docker/client"
	"github.com/moby/buildkit/client"
	"net"
)

func NewDockerClient(ctx context.Context, dockerClient dockerclient.CommonAPIClient) (*client.Client, error) {
	return client.New(ctx, "", client.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return dockerClient.DialHijack(ctx, "/grpc", "h2c", nil)
	}), client.WithSessionDialer(func(ctx context.Context, proto string, meta map[string][]string) (net.Conn, error) {
		return dockerClient.DialHijack(ctx, "/session", proto, meta)
	}))
}
