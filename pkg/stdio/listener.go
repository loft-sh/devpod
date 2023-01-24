package stdio

import "net"

// NewStdioListener creates a new stdio listener
func NewStdioListener() *StdioListener {
	return &StdioListener{
		connChan: make(chan net.Conn),
	}
}

// StdioListener implements the listener interface
type StdioListener struct {
	connChan chan net.Conn
}

// Ready implements interface
func (lis *StdioListener) Ready(conn net.Conn) {
	lis.connChan <- conn
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
