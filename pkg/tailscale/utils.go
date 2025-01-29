package tailscale

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

const LoftTSNetDomain = "ts.loft"

func GetHostname() (string, error) {
	osHostname, err := os.Hostname()
	if err != nil {
		fmt.Printf("Failed to get hostname: %v\n", err)
		return "", err
	}
	osHostname = strings.ToLower(strings.ReplaceAll(osHostname, ".", "-"))
	return fmt.Sprintf("devpod.%v.client", osHostname), nil
}

func GetURL(host string, port int) string {
	if port == 0 {
		return fmt.Sprintf("%s.%s", host, LoftTSNetDomain)
	}
	return fmt.Sprintf("%s.%s:%d", host, LoftTSNetDomain, port)
}

func DirectTunnel(ctx context.Context, network TSNet, host string, port int, stdin io.Reader, stdout io.Writer) error {
	address := fmt.Sprintf("%s.%s:%d", host, LoftTSNetDomain, port)
	conn, err := network.Dial(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server in proxy mode: %w", err)
	}
	defer conn.Close()

	// Forward stdin, stdout, and stderr for proxy mode
	go func() {
		_, _ = io.Copy(conn, stdin)
	}()
	go func() {
		_, _ = io.Copy(stdout, conn)
	}()

	// Block until the connection is closed
	<-ctx.Done()
	return nil
}
