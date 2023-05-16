package netstat

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/loft-sh/devpod/pkg/log"
)

type Forwarder interface {
	Forward(port string) error
	StopForward(port string) error
}

func NewWatcher(forwarder Forwarder, log log.Logger) *Watcher {
	return &Watcher{
		forwarder:      forwarder,
		forwardedPorts: map[string]bool{},
		log:            log,
	}
}

type Watcher struct {
	log log.Logger

	forwarder      Forwarder
	forwardedPorts map[string]bool
}

func (w *Watcher) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second * 3):
			err := w.runOnce()
			if err != nil {
				w.log.Errorf("Error watching ports: %v", err)
			}
		}
	}
}

func (w *Watcher) runOnce() error {
	newPorts, err := w.findPorts()
	if err != nil {
		return err
	}

	// stop ports that are not there anymore
	for port := range w.forwardedPorts {
		if !newPorts[port] {
			w.log.Debugf("Stop port %s", port)
			err = w.forwarder.StopForward(port)
			if err != nil {
				return fmt.Errorf("error stop forwarding port %s: %w", port, err)
			}
		}
	}

	// start ports that were not there before
	for port := range newPorts {
		if !w.forwardedPorts[port] {
			w.log.Debugf("Found open port %s ready to forward", port)
			err = w.forwarder.Forward(port)
			if err != nil {
				return fmt.Errorf("error forwarding port %s: %w", port, err)
			}
		}
	}

	w.forwardedPorts = newPorts
	return nil
}

func (w *Watcher) findPorts() (map[string]bool, error) {
	tcpSocks, err := TCPSocks(func(s *SockTabEntry) bool {
		return s.State == Listen
	})
	if err != nil {
		return nil, err
	}

	tcp6Socks, err := TCP6Socks(func(s *SockTabEntry) bool {
		return s.State == Listen
	})
	if err != nil {
		return nil, err
	}
	tcpSocks = append(tcpSocks, tcp6Socks...)

	// we only return ports that are within range 1024-12000 that have a program assigned
	retSocks := map[string]bool{}
	for _, sock := range tcpSocks {
		if sock.LocalAddr.Port < 1024 || sock.LocalAddr.Port > 12000 || sock.LocalAddr == nil {
			continue
		}

		retSocks[strconv.Itoa(int(sock.LocalAddr.Port))] = true
	}

	return retSocks, nil
}
