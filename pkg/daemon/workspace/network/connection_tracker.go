package network

import "sync"

// ConnTracker is a simple connection counter used by several services.
type ConnTracker struct {
	mu    sync.Mutex
	count int
}

func (c *ConnTracker) Add() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
}

func (c *ConnTracker) Remove() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count--
}

func (c *ConnTracker) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}
