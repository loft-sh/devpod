package network

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/loft-sh/log"
	"tailscale.com/client/tailscale"
	"tailscale.com/tsnet"
)

// TODO: this is no longer needed since we have a generic network socket available.
// PlatformGitCredentialsService handles the /git-credentials endpoint.
type PlatformGitCredentialsService struct {
	listener      net.Listener
	config        *WorkspaceServerConfig
	tsServer      *tsnet.Server
	lc            *tailscale.LocalClient
	projectName   string
	workspaceName string
	log           log.Logger
}

// NewPlatformGitCredentialsService creates a new PlatformGitCredentialsService.
func NewPlatformGitCredentialsService(config *WorkspaceServerConfig, tsServer *tsnet.Server, lc *tailscale.LocalClient, projectName, workspaceName string, log log.Logger) (*PlatformGitCredentialsService, error) {
	runnerProxySocket := filepath.Join(config.RootDir, RunnerProxySocket)
	_ = os.Remove(runnerProxySocket)
	l, err := net.Listen("unix", runnerProxySocket)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on socket %s: %w", runnerProxySocket, err)
	}
	_ = os.Chmod(runnerProxySocket, 0777)
	return &PlatformGitCredentialsService{
		listener:      l,
		config:        config,
		tsServer:      tsServer,
		lc:            lc,
		projectName:   projectName,
		workspaceName: workspaceName,
		log:           log,
	}, nil
}

// Start begins serving the /git-credentials endpoint.
func (s *PlatformGitCredentialsService) Start(ctx context.Context) {
	s.log.Infof("Starting Git Credentials server on %s", RunnerProxySocket)
	mux := http.NewServeMux()
	mux.HandleFunc("/git-credentials", s.gitCredentialsHandler)
	go func() {
		if err := http.Serve(s.listener, mux); err != nil && err != http.ErrServerClosed {
			s.log.Errorf("PlatformGitCredentialsService error: %v", err)
		}
	}()
}

func (s *PlatformGitCredentialsService) gitCredentialsHandler(w http.ResponseWriter, r *http.Request) {
	s.log.Infof("PlatformGitCredentialsService: received git credentials request from %s", r.RemoteAddr)
	discoveredRunner, err := discoverRunner(r.Context(), s.lc, s.log)
	if err != nil {
		http.Error(w, "failed to discover runner", http.StatusInternalServerError)
		return
	}
	runnerURL := fmt.Sprintf("http://%s.ts.loft/devpod/%s/%s/workspace-git-credentials", discoveredRunner, s.projectName, s.workspaceName)
	parsedURL, err := url.Parse(runnerURL)
	if err != nil {
		http.Error(w, "failed to parse runner URL", http.StatusInternalServerError)
		return
	}
	proxy := newReverseProxy(parsedURL, func(h http.Header) {
		h.Set("Authorization", "Bearer "+s.config.AccessKey)
	})
	transport := &http.Transport{DialContext: s.tsServer.Dial}
	proxy.Transport = transport
	proxy.ServeHTTP(w, r)
}

// Stop stops the PlatformGitCredentialsService by closing its listener.
func (s *PlatformGitCredentialsService) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
}
