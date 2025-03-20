package ts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/loft-sh/log"
	"tailscale.com/client/tailscale"
	"tailscale.com/ipn"
	"tailscale.com/types/netmap"
)

const LoftTSNetDomain = "ts.loft"

func GetClientHostname(userName string) (string, error) {
	osHostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	osHostname = strings.ToLower(strings.ReplaceAll(osHostname, ".", "-"))
	return fmt.Sprintf("devpod.%s.%s.client", osHostname, userName), nil
}

func GetWorkspaceHostname(name, namespace string) string {
	return fmt.Sprintf("devpod.%s.%s.workspace", name, namespace)
}

func ParseWorkspaceHostname(hostname string) (name string, project string, err error) {
	parts := strings.SplitN(hostname, ".", 4)
	if len(parts) != 4 {
		return name, project, fmt.Errorf("invalid hostname: %s", hostname)
	}

	name = parts[1]
	project = parts[2]

	return name, project, nil
}

func GetURL(host string, port int) string {
	if port == 0 {
		return fmt.Sprintf("%s.%s", host, LoftTSNetDomain)
	}
	return fmt.Sprintf("%s.%s:%d", host, LoftTSNetDomain, port)
}

// WaitHostReachable polls until the given host is reachable via ts.
func WaitHostReachable(ctx context.Context, lc *tailscale.LocalClient, addr Addr, maxRetries int, log log.Logger) error {
	for i := 0; i < maxRetries; i++ {
		timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		conn, err := lc.DialTCP(timeoutCtx, addr.Host(), uint16(addr.Port()))
		if err == nil {
			_ = conn.Close()
			return nil // Host is reachable
		}
		log.Debugf("Host %s not reachable, retrying... (%d/%d)", addr.String(), i+1, maxRetries)
		time.Sleep(200 * time.Millisecond)

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return fmt.Errorf("host %s not reachable", addr.String())
}

func WatchNetmap(ctx context.Context, lc *tailscale.LocalClient, netmapChangedFn func(nm *netmap.NetworkMap)) error {
	watcher, err := lc.WatchIPNBus(ctx, ipn.NotifyInitialNetMap|ipn.NotifyRateLimit|ipn.NotifyWatchEngineUpdates)
	if err != nil {
		return err
	}
	defer watcher.Close()

	var netMap *netmap.NetworkMap
	for {
		n, err := watcher.Next()
		if err != nil {
			return fmt.Errorf("watch ipn: %w", err)
		}
		if n.ErrMessage != nil {
			return fmt.Errorf("tailscale error: %w", errors.New(*n.ErrMessage))
		}
		if n.NetMap != nil {
			if n.NetMap != netMap {
				netMap = n.NetMap
				netmapChangedFn(netMap)
			}
		}
	}
}
