package ts

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type ConnTrackingFunc func(address string)

// CheckDerpConnection validates the DERP connection
func CheckDerpConnection(ctx context.Context, baseUrl *url.URL) error {
	newTransport := http.DefaultTransport.(*http.Transport).Clone()
	newTransport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	client := &http.Client{
		Transport: newTransport,
		Timeout:   5 * time.Second,
	}

	derpUrl := *baseUrl
	derpUrl.Path = "/derp/probe"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, derpUrl.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	res, err := client.Do(req)
	if err != nil || (res != nil && res.StatusCode != http.StatusOK) {
		return fmt.Errorf("failed to reach the coordinator server: %w", err)
	}

	return nil
}

// Utility function to get environment variable or default
func GetEnvOrDefault(envVar, defaultVal string) string {
	if val := os.Getenv(envVar); val != "" {
		return val
	}
	return defaultVal
}

// RemoveProtocol removes protocol from URL
func RemoveProtocol(hostPath string) string {
	if idx := strings.Index(hostPath, "://"); idx != -1 {
		return hostPath[idx+3:]
	}
	return hostPath
}
