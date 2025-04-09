package workspace

import (
	"context"
	"fmt"
	"os"

	"sync"

	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
)

// RunNetworkServer starts the network server.
func RunNetworkServer(ctx context.Context, d *Daemon, errChan chan<- error, wg *sync.WaitGroup, rootDir string) {
	defer wg.Done()
	if err := os.MkdirAll(rootDir, os.ModePerm); err != nil {
		errChan <- err
		return
	}
	logger := initLogging()
	config := client.NewConfig()
	config.AccessKey = d.Config.Platform.AccessKey
	config.Host = "https://" + d.Config.Platform.PlatformHost
	config.Insecure = true
	baseClient := client.NewClientFromConfig(config)
	if err := baseClient.RefreshSelf(ctx); err != nil {
		errChan <- fmt.Errorf("failed to refresh client: %w", err)
		return
	}
	tsServer := NewWorkspaceServer(&WorkspaceServerConfig{
		AccessKey:     d.Config.Platform.AccessKey,
		PlatformHost:  ts.RemoveProtocol(d.Config.Platform.PlatformHost),
		WorkspaceHost: d.Config.Platform.WorkspaceHost,
		Client:        baseClient,
		RootDir:       rootDir,
		LogF: func(format string, args ...interface{}) {
			logger.Infof(format, args...)
		},
	}, logger)
	if err := tsServer.Start(ctx); err != nil {
		errChan <- fmt.Errorf("network server: %w", err)
	}
}

func initLogging() log.Logger {
	return log.NewStdoutLogger(nil, os.Stdout, os.Stderr, logrus.InfoLevel)
}
