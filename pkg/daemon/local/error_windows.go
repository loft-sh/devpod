//go:build windows

package local

import (
	"errors"

	"golang.org/x/sys/windows"
)

func isConnectToDaemonError(err error) bool {
	return errors.Is(err, windows.WSAECONNREFUSED)
}
