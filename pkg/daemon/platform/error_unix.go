//go:build linux || darwin || unix

package daemon

import (
	"errors"
	"syscall"
)

func isConnectToDaemonError(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ENOENT)
}
