package client

import (
	"sync"
)

func newEventBuffer(size int) *eventBuffer {
	return &eventBuffer{
		bufferSize: size,
		buffer:     make([]Event, 0, size),
		fullChan:   make(chan struct{}),
	}
}

type eventBuffer struct {
	m          sync.Mutex
	bufferSize int
	buffer     []Event

	fullOnce sync.Once
	fullChan chan struct{}
}

func (e *eventBuffer) Drain() []Event {
	e.m.Lock()
	defer e.m.Unlock()

	e.close()
	return e.buffer
}

func (e *eventBuffer) Full() <-chan struct{} {
	return e.fullChan
}

func (e *eventBuffer) IsFull() bool {
	e.m.Lock()
	defer e.m.Unlock()

	return len(e.buffer) >= e.bufferSize
}

func (e *eventBuffer) Append(ev Event) bool {
	e.m.Lock()
	defer e.m.Unlock()

	// add to buffer if below capacity
	wasAdded := false
	if len(e.buffer) < e.bufferSize {
		e.buffer = append(e.buffer, ev)
		wasAdded = true
	}

	// we drop the event here if buffer is full
	if len(e.buffer) >= e.bufferSize {
		e.close()
	}

	return wasAdded
}

func (e *eventBuffer) close() {
	e.fullOnce.Do(func() {
		close(e.fullChan)
	})
}
