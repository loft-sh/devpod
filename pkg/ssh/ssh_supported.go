//go:build !windows

package ssh

import (
	"context"
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
)

func WatchWindowSize(ctx context.Context) <-chan os.Signal {
	windowSize := make(chan os.Signal, 1)
	signal.Notify(windowSize, unix.SIGWINCH)
	go func() {
		<-ctx.Done()
		signal.Stop(windowSize)
	}()
	return windowSize
}
