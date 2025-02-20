//go:build windows

package daemon

import (
	"fmt"
	"gopkg.in/natefinch/npipe.v2"
	"net"
	"time"
)

func GetSocketAddr(preferredDir, providerName string) string {
	return fmt.Sprintf("\\\\.\\pipe\\devpod.%s", providerName)
}

func dial(addr string) (net.Conn, error) {
	return npipe.DialTimeout(addr, 5*time.Second)
}

func listen(addr string) (net.Listener, error) {
	return npipe.Listen(addr)
}
