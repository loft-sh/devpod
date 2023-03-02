package stdio

import (
	"io"
	"net"
	"os"
	"time"
)

// StdioStream is the struct that implements the net.Conn interface
type StdioStream struct {
	in     io.Reader
	out    io.WriteCloser
	local  *StdinAddr
	remote *StdinAddr

	exitOnClose bool
}

// NewStdioStream is used to implement the connection interface
func NewStdioStream(in io.Reader, out io.WriteCloser, exitOnClose bool) *StdioStream {
	return &StdioStream{
		local:       NewStdinAddr("local"),
		remote:      NewStdinAddr("remote"),
		in:          in,
		out:         out,
		exitOnClose: exitOnClose,
	}
}

// LocalAddr implements interface
func (s *StdioStream) LocalAddr() net.Addr {
	return s.local
}

// RemoteAddr implements interface
func (s *StdioStream) RemoteAddr() net.Addr {
	return s.remote
}

// Read implements interface
func (s *StdioStream) Read(b []byte) (n int, err error) {
	return s.in.Read(b)
}

// Write implements interface
func (s *StdioStream) Write(b []byte) (n int, err error) {
	return s.out.Write(b)
}

// Close implements interface
func (s *StdioStream) Close() error {
	if s.exitOnClose {
		// We kill ourself here because the streams are closed
		os.Exit(0)
	}

	return s.out.Close()
}

// SetDeadline implements interface
func (s *StdioStream) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline implements interface
func (s *StdioStream) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline implements interface
func (s *StdioStream) SetWriteDeadline(t time.Time) error {
	return nil
}

// StdinAddr is the struct for the stdi
type StdinAddr struct {
	s string
}

// NewStdinAddr creates a new StdinAddr
func NewStdinAddr(s string) *StdinAddr {
	return &StdinAddr{s}
}

// Network implements interface
func (a *StdinAddr) Network() string {
	return "stdio"
}

func (a *StdinAddr) String() string {
	return a.s
}
