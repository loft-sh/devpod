package ts

import "net"

var _ net.Addr = Addr{}

type Addr struct {
	host string
	port int
}

func NewAddr(rawHost string, port int) Addr {
	return Addr{host: rawHost, port: port}
}

func (a Addr) Network() string {
	return "tcp"
}

func (a Addr) String() string {
	return GetURL(a.host, a.port)
}

func (a Addr) Host() string {
	return GetURL(a.host, 0)
}

func (a Addr) Port() int {
	return a.port
}
