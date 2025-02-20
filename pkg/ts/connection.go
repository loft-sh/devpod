package ts

import (
	"context"
	"sync"
	"time"

	"github.com/loft-sh/log"
)

func newConnectionCounter(ctx context.Context, timeout time.Duration, onConnect, onDisconnect ConnTrackingFunc, log log.Logger) *connectionCounter {
	return &connectionCounter{
		conns:        map[string]int{},
		generations:  map[string]int{},
		timeout:      timeout,
		onConnect:    onConnect,
		onDisconnect: onDisconnect,
		log:          log,
		ctx:          ctx,
	}
}

type connectionCounter struct {
	conns       map[string]int
	generations map[string]int
	m           sync.Mutex

	ctx          context.Context
	timeout      time.Duration
	onConnect    ConnTrackingFunc
	onDisconnect ConnTrackingFunc
	log          log.Logger
}

func (c *connectionCounter) Add(address string) {
	c.m.Lock()
	defer c.m.Unlock()

	c.conns[address]++
	c.log.Debugf("New connection on %s (Total: %d)", address, c.conns[address])
	c.onConnect(address)
}

func (c *connectionCounter) Dec(address string) {
	c.m.Lock()
	defer c.m.Unlock()

	c.conns[address]--
	connCount := c.conns[address]
	c.log.Debugf("Closed connection on %s (Total: %d)", address, connCount)

	if connCount <= 0 && c.timeout > 0 {
		c.generations[address]++

		go func(generation int, address string) {
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(c.timeout):
				c.m.Lock()
				defer c.m.Unlock()

				if c.generations[address] == generation && c.conns[address] <= 0 {
					c.onDisconnect(address)
				}
			}
		}(c.generations[address], address)
	}
}
