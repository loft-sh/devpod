package open

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/log"
	"github.com/skratchdot/open-golang/open"
)

// Open opens the given url in the default application, retrying every second until the context is done
func Open(ctx context.Context, url string, log log.Logger) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
			err := tryOpen(ctx, url, open.Start, log)
			if err == nil {
				return nil
			}
		}
	}
}

// JLabDesktop opens the given url in the JLab desktop application, retrying every second until the context is done
func JLabDesktop(ctx context.Context, url string, log log.Logger) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
			err := tryOpen(ctx, url, jlabOpen, log)
			if err == nil {
				return nil
			}
		}
	}
}

func jlabOpen(url string) error {
	return exec.Command("jlab", url).Run()
}

func tryOpen(ctx context.Context, url string, fn func(string) error, log log.Logger) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := devpodhttp.GetHTTPClient().Do(req)
	if err != nil {
		return err
	}

	if resp != nil {
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadGateway && resp.StatusCode != http.StatusServiceUnavailable {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Second):
			}
			_ = fn(url)
			log.Donef("Successfully opened %s", url)
			return nil
		}
	}

	return fmt.Errorf("not reachable")
}
