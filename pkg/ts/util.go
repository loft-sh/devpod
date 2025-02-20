package ts

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/loft-sh/log"
	"tailscale.com/client/tailscale"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
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

func DirectTunnel(ctx context.Context, lc *tailscale.LocalClient, host string, port int, stdin io.Reader, stdout io.Writer) error {
	conn, err := lc.DialTCP(ctx, host, uint16(port))
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server in proxy mode: %w", err)
	}
	defer conn.Close()

	errChan := make(chan error, 1)
	go func() {
		_, err := io.Copy(stdout, conn)
		errChan <- err
	}()
	go func() {
		_, err := io.Copy(conn, stdin)
		errChan <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

func WaitNodeReady(ctx context.Context, lc *tailscale.LocalClient) error {
	watcher, err := lc.WatchIPNBus(ctx, ipn.NotifyInitialState)
	if err != nil {
		return err
	}
	defer watcher.Close()

	for {
		// TODO: Improve this, checkout tailscales cli/up.go
		n, err := watcher.Next()
		if err != nil {
			return err
		}
		if n.ErrMessage != nil {
			return fmt.Errorf(*n.ErrMessage)
		}

		if s := n.State; s != nil && s.String() == ipn.Running.String() {
			return nil
		}

		fmt.Println("IPN message:", n)
	}
}

// TODO: Improve error handling
func CheckLocalNodeReady(status *ipnstate.Status) error {
	switch status.BackendState {
	case ipn.Stopped.String():
		return fmt.Errorf("Tailscale is stopped")
	case ipn.NeedsLogin.String():
		s := "Logged out."
		if status.AuthURL != "" {
			s += fmt.Sprintf("\nLog in at: %s", status.AuthURL)
		}
		return fmt.Errorf(s)
	case ipn.NeedsMachineAuth.String():
		return fmt.Errorf("Machine is not yet approved by tailnet admin")
	case ipn.Running.String(), ipn.Starting.String():
		return nil
	default:
		return fmt.Errorf("unexpected state: %s", status.BackendState)
	}
}

// WaitHostReachable polls until the given host is reachable via ts.
func WaitHostReachable(ctx context.Context, lc *tailscale.LocalClient, addr Addr, log log.Logger) error {
	const maxRetries = 20

	for i := 0; i < maxRetries; i++ {
		timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		conn, err := lc.DialTCP(timeoutCtx, addr.Host(), uint16(addr.Port()))
		if err == nil {
			_ = conn.Close()
			return nil // Host is reachable
		}
		log.Debugf("Host %s not reachable, retrying... (%d/%d)", addr.String(), i+1, maxRetries)

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return fmt.Errorf("host %s not reachable after %d attempts", addr.String(), maxRetries)
}
