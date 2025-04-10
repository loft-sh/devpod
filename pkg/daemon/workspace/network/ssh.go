package network

import (
	"context"
	"fmt"
	"io"
	"net"

	sshServer "github.com/loft-sh/devpod/pkg/ssh/server"
	"github.com/loft-sh/log"
	"tailscale.com/tsnet"
)

// SSHService handles SSH connections.
type SSHService struct {
	listener net.Listener
	tsServer *tsnet.Server
	log      log.Logger
	tracker  *ConnTracker
}

// NewSSHService creates a new SSHService.
func NewSSHService(tsServer *tsnet.Server, tracker *ConnTracker, log log.Logger) (*SSHService, error) {
	l, err := tsServer.Listen("tcp", fmt.Sprintf(":%d", sshServer.DefaultUserPort))
	if err != nil {
		return nil, fmt.Errorf("failed to listen for SSH on port %d: %w", sshServer.DefaultUserPort, err)
	}
	return &SSHService{
		listener: l,
		tsServer: tsServer,
		log:      log,
		tracker:  tracker,
	}, nil
}

// Start begins accepting SSH connections.
func (s *SSHService) Start(ctx context.Context) {
	s.log.Infof("Starting SSH listener")
	go s.acceptLoop(ctx)
}

func (s *SSHService) acceptLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		conn, err := s.listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			s.log.Errorf("SSHService: failed to accept connection: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *SSHService) handleConnection(conn net.Conn) {
	s.tracker.Add()
	defer s.tracker.Remove()
	defer conn.Close()

	localAddr := fmt.Sprintf("127.0.0.1:%d", sshServer.DefaultUserPort)
	backendConn, err := net.Dial("tcp", localAddr)
	if err != nil {
		s.log.Errorf("SSHService: failed to connect to local address %s: %v", localAddr, err)
		return
	}
	defer backendConn.Close()

	go func() {
		_, _ = io.Copy(backendConn, conn)
	}()
	_, _ = io.Copy(conn, backendConn)
}

// Stop stops the SSH server by closing its listener.
func (s *SSHService) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
}
