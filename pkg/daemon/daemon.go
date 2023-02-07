package daemon

import (
	"github.com/pkg/errors"
	"github.com/takama/daemon"
)

func InstallDaemon() error {
	service, err := daemon.New("devpod", "DevPod Agent Service", daemon.SystemDaemon)
	if err != nil {
		return err
	}

	// install ourselves with devpod watch
	_, err = service.Install("agent", "watch")
	if err != nil && err != daemon.ErrAlreadyInstalled {
		return errors.Wrap(err, "install service")
	}

	// make sure daemon is started
	_, err = service.Start()
	if err != nil && err != daemon.ErrAlreadyRunning {
		return errors.Wrap(err, "start service")
	}

	return nil
}
