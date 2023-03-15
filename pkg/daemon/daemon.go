package daemon

import (
	"fmt"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/pkg/errors"
	"github.com/takama/daemon"
	"runtime"
)

func InstallDaemon(agentDir string, log log.Logger) error {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return fmt.Errorf("unsupported daemon os")
	}

	// check if admin
	service, err := daemon.New("devpod", "DevPod Agent Service", daemon.SystemDaemon)
	if err != nil {
		return err
	}

	// install ourselves with devpod watch
	args := []string{"agent", "daemon"}
	if agentDir != "" {
		args = append(args, "--agent-dir", agentDir)
	}
	_, err = service.Install(args...)
	if err != nil && err != daemon.ErrAlreadyInstalled {
		return errors.Wrap(err, "install service")
	}

	// make sure daemon is started
	_, err = service.Start()
	if err != nil && err != daemon.ErrAlreadyRunning {
		return errors.Wrap(err, "start service")
	} else if err == nil {
		log.Infof("Successfully installed DevPod daemon into server")
	}

	return nil
}

func RemoveDaemon() error {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return fmt.Errorf("unsupported daemon os")
	}

	// check if admin
	service, err := daemon.New("devpod", "DevPod Agent Service", daemon.SystemDaemon)
	if err != nil {
		return err
	}

	// remove daemon
	_, err = service.Remove()
	if err != nil && err != daemon.ErrNotInstalled {
		return err
	}

	return nil
}
