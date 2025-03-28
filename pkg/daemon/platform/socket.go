//go:build linux || darwin || unix

package daemon

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

func GetSocketAddr(providerName string) string {
	return filepath.Join("/tmp", fmt.Sprintf("devpod-%s.sock", providerName))
}

func Dial(addr string) (net.Conn, error) {
	return net.DialTimeout("unix", addr, 2*time.Second)
}

func listen(addr string) (net.Listener, error) {
	conn, err := net.Dial("unix", addr)
	if err == nil {
		conn.Close()
		return nil, fmt.Errorf("%s: address already in use", addr)
	}
	_ = os.Remove(addr)

	sockDir := filepath.Dir(addr)
	if _, err := os.Stat(sockDir); errors.Is(err, os.ErrNotExist) {
		_ = os.MkdirAll(sockDir, 0o755) // best effort
	}
	pipe, err := net.Listen("unix", addr)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(addr, 0o666)
	return pipe, err
}
