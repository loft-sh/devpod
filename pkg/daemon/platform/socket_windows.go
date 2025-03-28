//go:build windows

package daemon

import (
	"fmt"
	"net"
	"time"

	"gopkg.in/natefinch/npipe.v2"
)

func GetSocketAddr(providerName string) string {
	return fmt.Sprintf("\\\\.\\pipe\\devpod.%s", providerName)
}

func Dial(addr string) (net.Conn, error) {
	return npipe.DialTimeout(addr, 2*time.Second)
}

func listen(addr string) (net.Listener, error) {
	return npipe.Listen(addr)
}
