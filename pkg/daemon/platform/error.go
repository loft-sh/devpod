package daemon

import (
	"errors"
	"fmt"
	"strings"

	"github.com/loft-sh/devpod/pkg/platform/client"
)

type DaemonNotAvailableError struct {
	Err      error
	Provider string
}

func (e *DaemonNotAvailableError) Error() string {
	return fmt.Sprintf("The DevPod Daemon for provider %s isn't reachable. Is DevPod Desktop or `devpod pro daemon start --host=$YOUR_PRO_HOST` running? %v", e.Provider, e.Err)
}
func (e *DaemonNotAvailableError) Unwrap() error {
	return e.Err
}

func IsDaemonNotAvailableError(err error) bool {
	var e *DaemonNotAvailableError
	return errors.As(err, &e)
}

func IsAccessKeyNotFound(err error) bool {
	// we have to check against the string because the error is coming from the server
	return strings.Contains(err.Error(), client.ErrAccessKeyNotFound.Error())
}
