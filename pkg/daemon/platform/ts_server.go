package daemon

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"tailscale.com/client/tailscale"
	"tailscale.com/envknob"
	"tailscale.com/ipn/store"
	"tailscale.com/tsnet"
	"tailscale.com/types/logger"
)

func newTSServer(ctx context.Context, host, accessKey, userName, rootDir string, insecure bool, log log.Logger) (*tsnet.Server, *tailscale.LocalClient, error) {
	// Build the platform URL
	baseUrl := url.URL{
		Scheme: ts.GetEnvOrDefault("LOFT_TSNET_SCHEME", "https"),
		Host:   ts.RemoveProtocol(host),
	}
	if err := ts.CheckDerpConnection(ctx, &baseUrl); err != nil {
		return nil, nil, fmt.Errorf("failed to verify DERP connection: %w", err)
	}
	if insecure {
		envknob.Setenv("TS_DEBUG_TLS_DIAL_INSECURE_SKIP_VERIFY", "true")
	}
	hostname, err := ts.GetClientHostname(userName)
	if err != nil {
		return nil, nil, fmt.Errorf("get hostname: %w", err)
	}
	statePath := filepath.Join(rootDir, provider.DaemonStateFile)
	store, err := store.NewFileStore(logger.Discard, statePath)
	if err != nil {
		return nil, nil, fmt.Errorf("new state store: %w", err)
	}

	logPrefix := "[ts] "
	logf := func(format string, args ...any) {
		if log.GetLevel() == logrus.DebugLevel {
			log.Debugf(logPrefix+format, args...)
		}
	}
	userLogf := func(format string, args ...any) {
		log.Infof(logPrefix+format, args...)
	}

	server := &tsnet.Server{
		Hostname:   hostname,
		Logf:       logf,
		UserLogf:   userLogf,
		ControlURL: baseUrl.String() + "/coordinator/",
		AuthKey:    accessKey,
		Dir:        rootDir,
		Ephemeral:  true,
		Store:      store,
	}

	lc, err := server.LocalClient()
	if err != nil {
		return nil, nil, err
	}

	return server, lc, nil
}
