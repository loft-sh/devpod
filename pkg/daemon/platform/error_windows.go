//go:build windows

package daemon

import (
	"errors"

	"golang.org/x/sys/windows"
)

func isConnectToDaemonError(err error) bool {
	return errors.Is(err, windows.WSAECONNREFUSED)
}
