package ts

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/log"
	"k8s.io/klog/v2"
)

// checkDerpConnection validates the DERP connection
func checkDerpConnection(ctx context.Context, baseUrl *url.URL) error {
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
		klog.FromContext(ctx).Error(err, "Failed to reach the coordinator server.", "url", derpUrl.String())

		if res != nil {
			body, _ := io.ReadAll(res.Body)
			defer res.Body.Close()
			klog.FromContext(ctx).V(1).Info("Response body", "body", string(body))
		}

		return fmt.Errorf("failed to reach the coordinator server: %w", err)
	}

	return nil
}

// Utility function to get environment variable or default
func getEnvOrDefault(envVar, defaultVal string) string {
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

func newConnectionCounter(ctx context.Context, timeout time.Duration, onTimeout func(address string), log log.Logger) *connectionCounter {
	return &connectionCounter{
		conns:       map[string]int{},
		generations: map[string]int{},
		timeout:     timeout,
		onTimeout:   onTimeout,
		log:         log,
		ctx:         ctx,
	}
}

type connectionCounter struct {
	conns       map[string]int
	generations map[string]int
	m           sync.Mutex

	ctx       context.Context
	timeout   time.Duration
	onTimeout func(address string)
	log       log.Logger
}

func (c *connectionCounter) Add(address string) {
	c.m.Lock()
	defer c.m.Unlock()

	c.conns[address]++
	c.log.Debugf("New connection on %s (Total: %d)", address, c.conns[address])
}

func (c *connectionCounter) Dec(address string) {
	c.m.Lock()
	defer c.m.Unlock()

	c.conns[address]--
	connCount := c.conns[address]
	c.log.Debugf("Closed connection on %s (Total: %d)", address, connCount)

	if connCount <= 0 && c.timeout > 0 {
		c.generations[address]++

		go func(generation int, address string) {
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(c.timeout):
				c.m.Lock()
				defer c.m.Unlock()

				if c.generations[address] == generation && c.conns[address] <= 0 {
					c.onTimeout(address)
				}
			}
		}(c.generations[address], address)
	}
}
