package network

import (
	"net"
	"net/http"
	"path/filepath"
	"time"
)

// GetClient returns a new HTTP client that uses a DevPod network socket for communication.
func GetClient() *http.Client {
	// Set up HTTP transport that uses our network socket.
	socketPath := filepath.Join(RootDir, NetworkProxySocket)
	transport := &http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second, // TODO: extract this to config
	}

	return client
}
