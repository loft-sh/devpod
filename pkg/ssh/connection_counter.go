package ssh

import (
	"context"
	"sync"
	"time"

	"github.com/loft-sh/log"
)

func newConnectionCounter(ctx context.Context, timeout time.Duration, onTimeout func(), address string, log log.Logger) *connectionCounter {
	return &connectionCounter{
		ctx:       ctx,
		address:   address,
		timeout:   timeout,
		onTimeout: onTimeout,
		log:       log,
	}
}

type connectionCounter struct {
	address string

	ctx       context.Context
	timeout   time.Duration
	onTimeout func()
	log       log.Logger

	m           sync.Mutex
	connections int
	generation  int
}

func (c *connectionCounter) Add() {
	c.m.Lock()
	defer c.m.Unlock()

	c.connections++
	c.log.Debugf("New connection on %s (Total: %d)", c.address, c.connections)
}

func (c *connectionCounter) Dec() {
	c.m.Lock()
	defer c.m.Unlock()

	c.connections--
	c.log.Debugf("Closed connection on %s (Total: %d)", c.address, c.connections)
	if c.connections <= 0 && c.timeout > 0 {
		c.generation++

		go func(generation int) {
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(c.timeout):
				c.m.Lock()
				defer c.m.Unlock()

				if c.generation == generation && c.connections <= 0 {
					c.onTimeout()
				}
			}
		}(c.generation)
	}
}
