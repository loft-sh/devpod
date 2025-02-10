package tunnel

import (
	"context"
	"sync"

	"github.com/loft-sh/devpod/pkg/netstat"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/log"
	"golang.org/x/crypto/ssh"
)

// newForwarder returns a new forwarder using an SSH client and list of ports to forward,
// for each port a new go routine is used to manage the SSH channel
func newForwarder(sshClient *ssh.Client, forwardedPorts []string, log log.Logger) netstat.Forwarder {
	return &forwarder{
		sshClient:      sshClient,
		forwardedPorts: forwardedPorts,
		portMap:        map[string]context.CancelFunc{},
		log:            log,
	}
}

// forwarder multiplexes a SSH client to forward ports to the remote container
type forwarder struct {
	sync.Mutex

	sshClient      *ssh.Client
	forwardedPorts []string

	portMap map[string]context.CancelFunc
	log     log.Logger
}

// Forward opens an SSH channel in the existing connection with channel type "direct-tcpip" to forward the local port
func (f *forwarder) Forward(port string) error {
	f.Lock()
	defer f.Unlock()

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

// StopForward stops the port forwarding for the given port
func (f *forwarder) StopForward(port string) error {
	f.Lock()
	defer f.Unlock()

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
