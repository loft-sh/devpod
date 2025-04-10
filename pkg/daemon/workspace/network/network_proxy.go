package network

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/loft-sh/log"
	"tailscale.com/tsnet"
)

// NetworkProxyService listens on a Unix socket and proxies requests to TSNet peers.
type NetworkProxyService struct {
	listener net.Listener
	tsServer *tsnet.Server
	log      log.Logger
}

// NewNetworkProxyService creates a new NetworkProxyService.
func NewNetworkProxyService(socketPath string, tsServer *tsnet.Server, log log.Logger) (*NetworkProxyService, error) {
	_ = os.Remove(socketPath)
	l, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on socket %s: %w", socketPath, err)
	}
	if err := os.Chmod(socketPath, 0777); err != nil {
		log.Errorf("failed to set socket permissions on %s: %v", socketPath, err)
	}
	log.Infof("TSProxyServer: network proxy listening on socket %s", socketPath)
	return &NetworkProxyService{
		listener: l,
		tsServer: tsServer,
		log:      log,
	}, nil
}

// Start begins accepting connections on the TS proxy socket.
func (s *NetworkProxyService) Start(ctx context.Context) {
	go func() {
		defer s.listener.Close()
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					s.log.Infof("TSProxyServer: listener shutting down (context cancelled)")
					return
				default:
					s.log.Errorf("TSProxyServer: error accepting connection: %v", err)
					continue
				}
			}
			go s.handleConnection(ctx, conn)
		}
	}()
}

func (s *NetworkProxyService) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		s.log.Errorf("TSProxyServer: failed to read HTTP request: %v", err)
		return
	}
	target := req.Host
	if target == "" {
		s.log.Errorf("TSProxyServer: HTTP request does not contain a Host header")
		return
	}
	if !strings.Contains(target, ":") {
		target = target + ":80"
	}
	s.log.Infof("TSProxyServer: proxying request to target %s", target)
	tsConn, err := s.tsServer.Dial(ctx, "tcp", target)
	if err != nil {
		s.log.Errorf("TSProxyServer: error dialing target %s: %v", target, err)
		return
	}
	defer tsConn.Close()
	if err := req.Write(tsConn); err != nil {
		s.log.Errorf("TSProxyServer: error forwarding request to target %s: %v", target, err)
		return
	}
	if _, err := io.Copy(conn, tsConn); err != nil {
		s.log.Errorf("TSProxyServer: error forwarding response from target: %v", err)
	}
}

// Stop stops the TSProxyServer by closing its listener.
func (s *NetworkProxyService) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
}
