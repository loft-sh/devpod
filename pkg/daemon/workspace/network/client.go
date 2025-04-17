package network

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"time"
)

// Dial returns a net.Conn to the network proxy socket.
func Dial() (net.Conn, error) {
	socketPath := filepath.Join(RootDir, NetworkProxySocket)
	return net.Dial("unix", socketPath)
}

// GetContextDialer returns ContextDialer interface function that uses our network socket.
func GetContextDialer() func(ctx context.Context, addr string) (net.Conn, error) {
	// The 'addr' argument passed by grpc.DialContext is ignored here,
	// as we always dial the fixed unix socket path.
	return func(ctx context.Context, _ string) (net.Conn, error) {
		conn, err := Dial()
		if err != nil {
			return nil, fmt.Errorf("failed to dial proxy socket: %w", err)
		}
		return conn, nil
	}
}

// GetHTTPTransport returns http.Transport that uses our network socket for HTTP requests.
func GetHTTPTransport() *http.Transport {
	// Set up HTTP transport that uses our network socket.
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := Dial()
			if err != nil {
				return nil, fmt.Errorf("failed to dial proxy socket via http transport: %w", err)
			}
			return conn, nil
		},
	}
}

// GetHTTPClient returns a new HTTP client that uses our network socket for transport.
func GetHTTPClient() *http.Client {
	return &http.Client{
		Transport: GetHTTPTransport(),
		Timeout:   30 * time.Second,
	}
}
