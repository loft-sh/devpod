package ts

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"path/filepath"

	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	tsClient "tailscale.com/client/tailscale"
	"tailscale.com/envknob"
	"tailscale.com/ipn/store"
	"tailscale.com/safesocket"
	"tailscale.com/tsnet"
	"tailscale.com/tsweb/varz"
	"tailscale.com/types/logger"
	"tailscale.com/util/clientmetric"
)

const DefaultDebugPort int = 12022

type Daemon struct {
	socketPath string
	statePath  string
	stateDir   string

	log log.Logger
}

func NewDaemon(rootDir string, _ log.Logger) *Daemon {
	socketPath := filepath.Join(rootDir, provider.DaemonSocket)
	statePath := filepath.Join(rootDir, provider.DaemonStateFile)
	logPath := filepath.Join(rootDir, "devpodd.log")
	log := log.NewFileLogger(logPath, logrus.DebugLevel) // FIXME: Add proper logging

	return &Daemon{
		socketPath: socketPath,
		statePath:  statePath,
		stateDir:   rootDir,

		log: log,
	}
}

// TODO: If access key changes we need to restart daemon
// TODO: Clean up dir handling
func (d *Daemon) Start(ctx context.Context, debug bool) error {
	d.log.Infof("Starting Daemon on socket: %s", d.socketPath)

	configPath := filepath.Join(filepath.Join(d.stateDir, ".."), "loft-config.json")
	baseClient, err := client.InitClientFromPath(ctx, configPath)
	if err != nil {
		return err
	}

	// TODO: Handle empty config

	userName := platform.GetUserName(baseClient.Self())
	if userName == "" {
		return fmt.Errorf("user name not set")
	}

	// TODO: Move to our own implementation
	ln, err := safesocket.Listen(d.socketPath)
	if err != nil {
		return err
	}
	// Build the platform URL
	baseUrl := url.URL{
		Scheme: getEnvOrDefault("LOFT_TSNET_SCHEME", "https"),
		Host:   RemoveProtocol(baseClient.Config().Host),
	}

	// Check DERP connection
	if err := checkDerpConnection(ctx, &baseUrl); err != nil {
		return fmt.Errorf("failed to verify DERP connection: %w", err)
	}

	if baseClient.Config().Insecure {
		envknob.Setenv("TS_DEBUG_TLS_DIAL_INSECURE_SKIP_VERIFY", "true")
	}
	hostname, err := GetClientHostname(userName)
	if err != nil {
		return fmt.Errorf("get hostname: %w", err)
	}
	store, err := store.NewFileStore(logger.Discard, d.statePath)
	if err != nil {
		return err
	}
	tsServer := &tsnet.Server{
		Hostname: hostname,
		// Logf:       logger.Discard,
		ControlURL: baseUrl.String() + "/coordinator/",
		AuthKey:    baseClient.Config().AccessKey,
		Dir:        d.stateDir,
		Ephemeral:  false,
		Store:      store,
	}
	lc, err := tsServer.LocalClient()
	if err != nil {
		return err
	}

	forward(ctx, ln, lc)

	// if debug {
	// 	addr := net.JoinHostPort("localhost", strconv.Itoa(DefaultDebugPort))
	// 	debugServer := getDebugServer(addr)
	// 	go func() {
	// 		err := debugServer.ListenAndServe()
	// 		if err != nil {
	// 			d.log.Fatal(err)
	// 		}
	// 	}()
	// }

	return nil
}

func getDebugServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", serveHealth)
	mux.HandleFunc("/debug/metrics", servePrometheusMetrics)
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}

func servePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	varz.Handler(w, r)
	clientmetric.WritePrometheusExpositionFormat(w)
}

func serveHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func forward(ctx context.Context, ln net.Listener, lc *tsClient.LocalClient) {
	for {
		unixConn, err := ln.Accept()
		if err != nil {
			// TODO: logging
			continue
		}

		go func(conn net.Conn) {
			defer conn.Close()

			lcConn, err := lc.Dial(ctx, "tcp", "local-tailscaled.sock:80")
			if err != nil {
				// TODO: logging
				return
			}
			defer lcConn.Close()

			go func() {
				defer conn.Close()
				defer lcConn.Close()
				_, err = io.Copy(lcConn, conn)
			}()
			_, err = io.Copy(conn, lcConn)
		}(unixConn)
	}
}
