package stdio

import (
	"io"
	"net"
)

// NewStdioListener creates a new stdio listener
func NewStdioListener(reader io.Reader, writer io.WriteCloser, exitOnClose bool) *StdioListener {
	conn := NewStdioStream(reader, writer, exitOnClose)
	connChan := make(chan net.Conn)
	go func() {
		connChan <- conn
	}()

	return &StdioListener{
		connChan: connChan,
	}
}

// StdioListener implements the listener interface
type StdioListener struct {
	connChan chan net.Conn
}

// Ready implements interface
func (lis *StdioListener) Ready(conn net.Conn) {

}

// Accept implements interface
func (lis *StdioListener) Accept() (net.Conn, error) {
	return <-lis.connChan, nil
}

// Close implements interface
func (lis *StdioListener) Close() error {
	return nil
}

// Addr implements interface
func (lis *StdioListener) Addr() net.Addr {
	return NewStdinAddr("listener")
}
