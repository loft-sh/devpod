package network

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/loft-sh/log"
	"tailscale.com/client/tailscale"
	"tailscale.com/tsnet"
)

// HeartbeatService sends periodic heartbeats when there are active connections.
type HeartbeatService struct {
	tsServer      *tsnet.Server
	lc            *tailscale.LocalClient
	config        *WorkspaceServerConfig
	projectName   string
	workspaceName string
	log           log.Logger
	tracker       *ConnTracker
}

// NewHeartbeatService creates a new HeartbeatService.
func NewHeartbeatService(config *WorkspaceServerConfig, tsServer *tsnet.Server, lc *tailscale.LocalClient, projectName, workspaceName string, tracker *ConnTracker, log log.Logger) *HeartbeatService {
	return &HeartbeatService{
		tsServer:      tsServer,
		lc:            lc,
		config:        config,
		projectName:   projectName,
		workspaceName: workspaceName,
		log:           log,
		tracker:       tracker,
	}
}

// Start begins the heartbeat loop.
func (s *HeartbeatService) Start(ctx context.Context) {
	transport := &http.Transport{DialContext: s.tsServer.Dial}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if s.tracker.Count() > 0 {
				if err := s.sendHeartbeat(ctx, client); err != nil {
					s.log.Errorf("HeartbeatService: failed to send heartbeat: %v", err)
				}
			}
		}
	}
}

func (s *HeartbeatService) sendHeartbeat(ctx context.Context, client *http.Client) error {
	hbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	discoveredRunner, err := discoverRunner(hbCtx, s.lc, s.log)
	if err != nil {
		return fmt.Errorf("failed to discover runner: %w", err)
	}
	heartbeatURL := fmt.Sprintf("http://%s.ts.loft/devpod/%s/%s/heartbeat", discoveredRunner, s.projectName, s.workspaceName)
	s.log.Infof("HeartbeatService: sending heartbeat to %s, active connections: %d", heartbeatURL, s.tracker.Count())
	req, err := http.NewRequestWithContext(hbCtx, "GET", heartbeatURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %w", heartbeatURL, err)
	}
	req.Header.Set("Authorization", "Bearer "+s.config.AccessKey)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s failed: %w", heartbeatURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received response from %s - Status: %d", heartbeatURL, resp.StatusCode)
	}
	s.log.Infof("HeartbeatService: received response from %s - Status: %d", heartbeatURL, resp.StatusCode)
	return nil
}
