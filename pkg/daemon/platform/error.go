package daemon

import "fmt"

type errDaemonNotAvailable struct {
	Err      error
	Provider string
}

func (e errDaemonNotAvailable) Error() string {
	return fmt.Sprintf("The DevPod Daemon for provider %s isn't reachable. Is DevPod Desktop or `devpod pro daemon start --host=$YOUR_PRO_HOST` running? %v", e.Provider, e.Err)
}
