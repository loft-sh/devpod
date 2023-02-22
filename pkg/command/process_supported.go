//go:build linux || darwin || unix

package command

import (
	"os"
	"strconv"
	"syscall"
	"time"
)

func isRunning(pid string) (bool, error) {
	parsedPid, err := strconv.Atoi(pid)
	if err != nil {
		return false, err
	}

	process, err := os.FindProcess(parsedPid)
	if err != nil {
		return false, err
	}

	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return false, nil
	}

	return true, nil
}

func kill(pid string) error {
	parsedPid, err := strconv.Atoi(pid)
	if err != nil {
		return err
	}

	_ = syscall.Kill(parsedPid, syscall.SIGTERM)
	time.Sleep(2000)
	_ = syscall.Kill(parsedPid, syscall.SIGKILL)
	return nil
}
