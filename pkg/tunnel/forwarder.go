package tunnel

import (
	"context"
	"sync"

	"github.com/loft-sh/devpod/pkg/netstat"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/log"
	"golang.org/x/crypto/ssh"
)

func newForwarder(sshClient *ssh.Client, forwardedPorts []string, log log.Logger) netstat.Forwarder {
	return &forwarder{
		sshClient:      sshClient,
		forwardedPorts: forwardedPorts,
		portMap:        map[string]context.CancelFunc{},
		log:            log,
	}
}

type forwarder struct {
	m sync.Mutex

	sshClient      *ssh.Client
	forwardedPorts []string

	portMap map[string]context.CancelFunc
	log     log.Logger
}

func (f *forwarder) Forward(port string) error {
	f.m.Lock()
	defer f.m.Unlock()

	if f.isExcluded(port) || f.portMap[port] != nil {
		return nil
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	f.portMap[port] = cancel
	f.log.Infof("Start port-forwarding on port %s", port)

	go func(port string) {
		// do the forward
		err := devssh.PortForward(cancelCtx, f.sshClient, "tcp", "localhost:"+port, "tcp", "localhost:"+port, 0, f.log)
		if err != nil {
			f.log.Errorf("Error port forwarding %s: %v", port, err)
		}
	}(port)

	return nil
}

func (f *forwarder) StopForward(port string) error {
	f.m.Lock()
	defer f.m.Unlock()

	if f.isExcluded(port) || f.portMap[port] == nil {
		return nil
	}

	f.log.Infof("Stop port-forwarding on port %s", port)
	f.portMap[port]()
	delete(f.portMap, port)

	return nil
}

func (f *forwarder) isExcluded(port string) bool {
	for _, p := range f.forwardedPorts {
		if p == port {
			return true
		}
	}

	return false
}
