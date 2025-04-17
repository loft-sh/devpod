package workspace

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/pkg/errors"
)

func SetupActivityFile() error {
	if err := os.WriteFile(agent.ContainerActivityFile, nil, 0777); err != nil {
		return err
	}
	return os.Chmod(agent.ContainerActivityFile, 0777)
}

// RunTimeoutMonitor monitors the activity file and sends an error if the timeout is exceeded.
func RunTimeoutMonitor(ctx context.Context, duration time.Duration, errChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stat, err := os.Stat(agent.ContainerActivityFile)
			if err != nil {
				continue
			}
			if !stat.ModTime().Add(duration).After(time.Now()) {
				errChan <- errors.New("timeout reached, terminating daemon")
				return
			}
		}
	}
}
