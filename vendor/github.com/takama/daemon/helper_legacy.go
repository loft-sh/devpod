// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by
// license that can be found in the LICENSE file.

//+build !go1.8

package daemon

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Service constants
const (
	success = "\t\t\t\t\t[  \033[32mOK\033[0m  ]" // Show colored "OK"
	failed  = "\t\t\t\t\t[\033[31mFAILED\033[0m]" // Show colored "FAILED"
)

var (
	// ErrUnsupportedSystem appears if try to use service on system which is not supported by this release
	ErrUnsupportedSystem = errors.New("Unsupported system")

	// ErrRootPrivileges appears if run installation or deleting the service without root privileges
	ErrRootPrivileges = errors.New("You must have root user privileges. Possibly using 'sudo' command should help")

	// ErrAlreadyInstalled appears if service already installed on the system
	ErrAlreadyInstalled = errors.New("Service has already been installed")

	// ErrNotInstalled appears if try to delete service which was not been installed
	ErrNotInstalled = errors.New("Service is not installed")

	// ErrAlreadyRunning appears if try to start already running service
	ErrAlreadyRunning = errors.New("Service is already running")

	// ErrAlreadyStopped appears if try to stop already stopped service
	ErrAlreadyStopped = errors.New("Service has already been stopped")
)

// ExecPath tries to get executable path
func ExecPath() (string, error) {
	return execPath()
}

// Lookup path for executable file
func executablePath(name string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return execPath()
}

// Check root rights to use system service
func checkPrivileges() (bool, error) {

	if output, err := exec.Command("id", "-g").Output(); err == nil {
		if gid, parseErr := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 32); parseErr == nil {
			if gid == 0 {
				return true, nil
			}
			return false, ErrRootPrivileges
		}
	}
	return false, ErrUnsupportedSystem
}
