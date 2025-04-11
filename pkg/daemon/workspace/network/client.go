package network

import (
	"context"
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

// GetCOntextDialer returns ContextDialer interface function that uses our network socket.
func GetContextDialer() func(ctx context.Context, addr string) (net.Conn, error) {
	return func(ctx context.Context, addr string) (net.Conn, error) {
		return Dial()
	}
}

// GetHTTPTransport returns http.Transport that uses our network socket for HTTP requests.
func GetHTTPTransport() *http.Transport {
	// Set up HTTP transport that uses our network socket.
	return &http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			return Dial()
		},
	}
}

// GetClient returns a new HTTP client that uses our network socket for transport.
func GetHTTPClient() *http.Client {
	return &http.Client{
		Transport: GetHTTPTransport(),
		Timeout:   30 * time.Second,
	}
}
