package inject

import (
	"bytes"
	"io"
	"sync"
)

func newDelayedWriter(writer io.Writer) *delayedWriter {
	return &delayedWriter{writer: writer}
}

type delayedWriter struct {
	m       sync.Mutex
	started bool

	buffer bytes.Buffer
	writer io.Writer
}

func (d *delayedWriter) Buffer() []byte {
	d.m.Lock()
	defer d.m.Unlock()

	return d.buffer.Bytes()
}

func (d *delayedWriter) Start() {
	if d.writer == nil {
		return
	}

	d.m.Lock()
	defer d.m.Unlock()

	if d.started {
		return
	}

	d.started = true
	data := d.buffer.Bytes()
	if len(data) == 0 {
		return
	}

	_, _ = d.writer.Write(data)
}

func (d *delayedWriter) Write(p []byte) (n int, err error) {
	if d.writer == nil {
		return len(p), nil
	}

	d.m.Lock()
	defer d.m.Unlock()

	if d.started {
		return d.writer.Write(p)
	}
	return d.buffer.Write(p)
}
