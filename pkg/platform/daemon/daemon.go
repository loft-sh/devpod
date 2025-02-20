package daemon

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	devpodlog "github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"tailscale.com/client/tailscale"
	"tailscale.com/tsnet"
)

type Daemon struct {
	socketListener net.Listener
	tsServer       *tsnet.Server
	localServer    *localServer
	log            log.Logger
}

func Init(ctx context.Context, rootDir string, providerName string, debug bool) (*Daemon, error) {
	log := initLogging(rootDir, debug)

	socket := GetSocketAddr(rootDir, providerName)
	log.Infof("Starting Daemon on address: %s", socket)
	// listen to socket for early return  in case it's already in use
	socketListener, err := listen(socket)
	if err != nil {
		return nil, fmt.Errorf("listen on socket: %w", err)
	}

	loftConfigPath := filepath.Join(rootDir, "..", "loft-config.json")
	baseClient, err := client.InitClientFromPath(ctx, loftConfigPath)
	if err != nil {
		return nil, err
	}

	userName := platform.GetUserName(baseClient.Self())
	if userName == "" {
		return nil, fmt.Errorf("user name not set")
	}

	tsServer, lc, err := getTSServer(ctx, baseClient.Config(), userName, rootDir, log)
	if err != nil {
		return nil, fmt.Errorf("get tailscale server: %w", err)
	}

	localServer, err := getLocalServer(lc)
	if err != nil {
		return nil, fmt.Errorf("get local server: %w", err)
	}

	return &Daemon{
		socketListener: socketListener,
		tsServer:       tsServer,
		localServer:    localServer,
		log:            log,
	}, nil
}

func (d *Daemon) Start() error {
	errChan := make(chan error, 1)
	go func() {
		d.log.Infof("Starting local server: %s", d.localServer.Addr())
		err := d.localServer.ListenAndServe()
		errChan <- err
	}()
	go func() {
		d.log.Info("Start proxying connections")
		err := d.Listen(d.socketListener)
		errChan <- err
	}()
	return <-errChan
}

func (d *Daemon) Listen(ln net.Listener) error {
	lc, err := d.tsServer.LocalClient()
	if err != nil {
		return fmt.Errorf("get local tailscale client: %w", err)
	}

	for {
		rawConn, err := ln.Accept()
		if err != nil {
			d.log.Error("Failed to accept connection: %v", err)
			continue
		}
		d.log.Debug("Accepted connection")

		bConn := newBufferedConn(rawConn)
		clientType, err := getClientType(bConn)
		if err != nil {
			_ = bConn.Close()
			d.log.Debug("Failed to get client type: %w", err)
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
