package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	devpodlog "github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"tailscale.com/client/tailscale"
	"tailscale.com/tsnet"
	"tailscale.com/types/netmap"
)

type Daemon struct {
	socketListener net.Listener
	tsServer       *tsnet.Server
	localServer    *localServer
	rootDir        string
	log            log.Logger
}

type InitConfig struct {
	RootDir        string
	Context        string
	ProviderName   string
	UserName       string
	PlatformClient client.Client

	Debug bool
}

func Init(ctx context.Context, config InitConfig) (*Daemon, error) {
	log := initLogging(config.RootDir, config.Debug)

	socketAddr := GetSocketAddr(config.ProviderName)
	log.Infof("Starting Daemon on address: %s", socketAddr)
	// listen to socket for early return  in case it's already in use
	socketListener, err := listen(socketAddr)
	if err != nil {
		return nil, fmt.Errorf("listen on socket: %w", err)
	}

	platformConfig := config.PlatformClient.Config()
	tsServer, lc, err := newTSServer(ctx, platformConfig.Host, platformConfig.AccessKey, config.UserName, config.RootDir, platformConfig.Insecure, log)
	if err != nil {
		return nil, fmt.Errorf("create tailscale server: %w", err)
	}

	localServer, err := newLocalServer(lc, config.PlatformClient, config.Context, log)
	if err != nil {
		return nil, fmt.Errorf("create local server: %w", err)
	}

	return &Daemon{
		socketListener: socketListener,
		tsServer:       tsServer,
		localServer:    localServer,
		rootDir:        config.RootDir,
		log:            log,
	}, nil
}
func (d *Daemon) Start(ctx context.Context) error {
	errChan := make(chan error, 1)

	go func() {
		d.log.Infof("Starting local server: %s", d.localServer.Addr())
		errChan <- d.localServer.ListenAndServe()
	}()
	go func() {
		d.log.Info("Start proxying connections")
		errChan <- d.Listen(d.socketListener)
	}()
	go func() {
		d.log.Info("Start netmap watcher")
		errChan <- d.watchNetmap(ctx)
	}()

	defer func() {
		d.log.Info("Cleaning up daemon resources")
		_ = d.tsServer.Close()
		_ = d.localServer.Close()
		_ = d.socketListener.Close()
	}()

	select {
	case err := <-errChan:
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	case <-ctx.Done():
		err := ctx.Err()
		if !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	}
}

func (d *Daemon) Listen(ln net.Listener) error {
	lc, err := d.tsServer.LocalClient()
	if err != nil {
		return fmt.Errorf("get local tailscale client: %w", err)
	}

	for {
		rawConn, err := ln.Accept()
		if err != nil {
			d.log.Debugf("Failed to accept connection: %v", err)
			continue
		}

		bConn := newBufferedConn(rawConn)
		clientType, err := getClientType(bConn)
		if err != nil {
			_ = bConn.Close()
			d.log.Debugf("Unknown client type: %v", err)
			continue
		}
		switch clientType {
		case devPodClientType:
			go d.handler(bConn, dialLocal(d.localServer))
		case tailscaleClientType:
			go d.handler(bConn, dialTS(lc))
		}
	}
}

func (d *Daemon) handler(conn net.Conn, dialFunc dialFunc) {
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	backendConn, err := dialFunc(ctx)
	if err != nil {
		d.log.Error("dial: %v", err)
		return
	}
	defer backendConn.Close()

	errChan := make(chan error, 1)
	go func() {
		_, err := io.Copy(backendConn, conn)
		errChan <- err
	}()
	go func() {
		_, err := io.Copy(conn, backendConn)
		errChan <- err
	}()
	<-errChan
}

func (d *Daemon) watchNetmap(ctx context.Context) error {
	lc, err := d.tsServer.LocalClient()
	if err != nil {
		return err
	}

	return ts.WatchNetmap(ctx, lc, func(netMap *netmap.NetworkMap) {
		nm, err := json.Marshal(netMap)
		if err != nil {
			d.log.Errorf("Failed to marshal netmap: %v", err)
		} else {
			_ = os.WriteFile(filepath.Join(d.rootDir, "netmap.json"), nm, 0o644)
		}
	})
}

func initLogging(rootDir string, debug bool) log.Logger {
	logLevel := logrus.InfoLevel
	if debug {
		logLevel = logrus.DebugLevel
	}

	logPath := filepath.Join(rootDir, "daemon.log")
	logger := log.NewFileLogger(logPath, logLevel)
	if os.Getenv("DEVPOD_UI") != "true" {
		logger = devpodlog.NewCombinedLogger(logLevel, logger, log.NewStreamLogger(os.Stdout, os.Stderr, logLevel))
	}

	return logger
}

type dialFunc func(context.Context) (net.Conn, error)

func dialTS(lc *tailscale.LocalClient) dialFunc {
	return func(ctx context.Context) (net.Conn, error) {
		return lc.Dial(ctx, "tcp", "local-tailscaled.sock:80")
	}
}

func dialLocal(l *localServer) dialFunc {
	return func(ctx context.Context) (net.Conn, error) {
		return l.Dial(ctx, "tcp", l.Addr())
	}
}

type clientType string

var (
	devPodClientType    clientType = "devpod"
	tailscaleClientType clientType = "tailscale"
)

func getClientType(bConn *bufferedConn) (clientType, error) {
	b, err := bConn.ReadByte()
	if err != nil {
		return "", err
	}
	switch b {
	case devPodClientPrefix:
		return devPodClientType, nil
	default:
		return tailscaleClientType, bConn.UnreadByte()
	}
}

func newBufferedConn(conn net.Conn) *bufferedConn {
	return &bufferedConn{
		Conn: conn,
		br:   bufio.NewReader(conn),
	}
}

type bufferedConn struct {
	net.Conn
	br *bufio.Reader
}

func (c *bufferedConn) Read(b []byte) (int, error) {
	return c.br.Read(b)
}

func (c *bufferedConn) ReadByte() (byte, error) {
	return c.br.ReadByte()
}

func (c *bufferedConn) UnreadByte() error {
	return c.br.UnreadByte()
}
