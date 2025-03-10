package daemon

import (
	"fmt"
	"strings"

	"github.com/loft-sh/devpod/pkg/platform/client"
)

type daemonNotAvailableError struct {
	Err      error
	Provider string
}

func (e daemonNotAvailableError) Error() string {
	return fmt.Sprintf("The DevPod Daemon for provider %s isn't reachable. Is DevPod Desktop or `devpod pro daemon start --host=$YOUR_PRO_HOST` running? %v", e.Provider, e.Err)
}

func IsAccessKeyNotFound(err error) bool {
	// we have to check against the string because the error is coming from the server
	return strings.Contains(err.Error(), client.ErrAccessKeyNotFound.Error())
}
