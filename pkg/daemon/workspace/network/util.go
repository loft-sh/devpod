package network

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/loft-sh/log"
	"tailscale.com/client/tailscale"
)

// newReverseProxy creates a reverse proxy to the target and applies header modifications.
func newReverseProxy(target *url.URL, modifyHeaders func(http.Header)) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		dest := *target
		req.URL = &dest
		req.Host = dest.Host
		modifyHeaders(req.Header)
	}
	return proxy
}

// discoverRunner finds a peer whose hostname ends with "runner".
func discoverRunner(ctx context.Context, lc *tailscale.LocalClient, log log.Logger) (string, error) {
	status, err := lc.Status(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}
	var runner string
	for _, peer := range status.Peer {
		if peer == nil || peer.HostName == "" {
			continue
		}
		if strings.HasSuffix(peer.HostName, "runner") {
			runner = peer.HostName
			break
		}
	}
	if runner == "" {
		return "", fmt.Errorf("no active runner found")
	}
	log.Infof("discoverRunner: selected runner = %s", runner)
	return runner, nil
}
