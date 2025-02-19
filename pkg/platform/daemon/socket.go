//go:build linux || darwin || unix

package daemon

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

func listen(socket string) (net.Listener, error) {
	conn, err := net.Dial("unix", socket)
	if err == nil {
		conn.Close()
		return nil, fmt.Errorf("%s: address already in use", socket)
	}
	_ = os.Remove(socket)

	sockDir := filepath.Dir(socket)
	if _, err := os.Stat(sockDir); errors.Is(err, os.ErrNotExist) {
		_ = os.MkdirAll(sockDir, 0o755) // best effort
	}
	pipe, err := net.Listen("unix", socket)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(socket, 0o666)
	return pipe, err
}
