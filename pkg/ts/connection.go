package ts

import (
	"context"
	"sync"
	"time"

	"github.com/loft-sh/log"
)

type connectionCounter struct {
	conns       map[string]int
	generations map[string]int
	m           sync.Mutex

	ctx       context.Context
	timeout   time.Duration
	onTimeout func(address string)
	log       log.Logger
}

func newConnectionCounter(ctx context.Context, timeout time.Duration, onTimeout func(address string), log log.Logger) *connectionCounter {
	return &connectionCounter{
		conns:       map[string]int{},
		generations: map[string]int{},
		timeout:     timeout,
		onTimeout:   onTimeout,
		log:         log,
		ctx:         ctx,
	}
}

func (c *connectionCounter) Add(address string) {
	c.m.Lock()
	defer c.m.Unlock()

	c.conns[address]++
	c.log.Debugf("New connection on %s (Total: %d)", address, c.conns[address])
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
					c.onTimeout(address)
				}
			}
		}(c.generations[address], address)
	}
}
