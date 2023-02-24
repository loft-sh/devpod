package open

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/skratchdot/open-golang/open"
	"net/http"
	"time"
)

func Open(ctx context.Context, url string, log log.Logger) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
			err := tryOpen(ctx, url, log)
			if err == nil {
				return nil
			}
		}
	}
}

func tryOpen(ctx context.Context, url string, log log.Logger) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, "GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp != nil && resp.StatusCode != http.StatusBadGateway && resp.StatusCode != http.StatusServiceUnavailable {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
		}
		_ = open.Start(url)
		log.Donef("Successfully opened %s", url)
		return nil
	}

	return fmt.Errorf("not reachable")
}
