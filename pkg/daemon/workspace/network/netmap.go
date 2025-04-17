package network

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	"tailscale.com/client/tailscale"
	"tailscale.com/types/netmap"
)

// NetmapWatcherService watches the Tailscale netmap and writes it to a file.
type NetmapWatcherService struct {
	rootDir string
	lc      *tailscale.LocalClient
	log     log.Logger
}

// NewNetmapWatcherService creates a new NetmapWatcherService.
func NewNetmapWatcherService(rootDir string, lc *tailscale.LocalClient, log log.Logger) *NetmapWatcherService {
	return &NetmapWatcherService{
		rootDir: rootDir,
		lc:      lc,
		log:     log,
	}
}

// Start begins watching the netmap.
func (s *NetmapWatcherService) Start(ctx context.Context) {
	lastUpdate := time.Now()
	if err := ts.WatchNetmap(ctx, s.lc, func(netMap *netmap.NetworkMap) {
		if time.Since(lastUpdate) < netMapCooldown {
			return
		}
		lastUpdate = time.Now()
		nm, err := json.Marshal(netMap)
		if err != nil {
			s.log.Errorf("NetmapWatcherService: failed to marshal netmap: %v", err)
		} else {
			_ = os.WriteFile(filepath.Join(s.rootDir, "netmap.json"), nm, 0644)
		}
	}); err != nil {
		s.log.Errorf("NetmapWatcherService: failed to watch netmap: %v", err)
	}
}
