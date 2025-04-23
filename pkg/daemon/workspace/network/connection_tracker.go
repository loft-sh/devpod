package network

import (
	"sync"

	"github.com/loft-sh/log"
)

// ConnTracker is a simple connection counter used by several services.
type ConnTracker struct {
	mu    sync.Mutex
	count int

	logger log.Logger
}

func (c *ConnTracker) Add(serviceName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
	c.logger.Debugf("%s: Added new connection, connection count %d\n", serviceName, c.count)
}

func (c *ConnTracker) Remove(serviceName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count--
	c.logger.Debugf("%s: Removed connection, connection count %d\n", serviceName, c.count)
}

func (c *ConnTracker) Count(serviceName string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logger.Debugf("%s: Get connection count %d\n", serviceName, c.count)
	return c.count
}
