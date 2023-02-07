// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by
// license that can be found in the LICENSE file.

// Package daemon windows version
package daemon

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// windowsRecord - standard record (struct) for windows version of daemon package
type windowsRecord struct {
	name         string
	description  string
	kind         Kind
	dependencies []string
}

func newDaemon(name, description string, kind Kind, dependencies []string) (Daemon, error) {

	return &windowsRecord{name, description, kind, dependencies}, nil
}

// Install the service
func (windows *windowsRecord) Install(args ...string) (string, error) {
	installAction := "Install " + windows.description + ":"

	execp, err := execPath()

	if err != nil {
		return installAction + failed, err
	}

	m, err := mgr.Connect()
	if err != nil {
		return installAction + failed, err
	}
	defer m.Disconnect()

	s, err := m.OpenService(windows.name)
	if err == nil {
		s.Close()
		return installAction + failed, ErrAlreadyRunning
	}

	s, err = m.CreateService(windows.name, execp, mgr.Config{
		DisplayName:  windows.name,
		Description:  windows.description,
		StartType:    mgr.StartAutomatic,
		Dependencies: windows.dependencies,
	}, args...)
	if err != nil {
		return installAction + failed, err
	}
	defer s.Close()

	// set recovery action for service
	// restart after 5 seconds for the first 3 times
	// restart after 1 minute, otherwise
	r := []mgr.RecoveryAction{
		mgr.RecoveryAction{
			Type:  mgr.ServiceRestart,
			Delay: 5000 * time.Millisecond,
		},
		mgr.RecoveryAction{
			Type:  mgr.ServiceRestart,
			Delay: 5000 * time.Millisecond,
		},
		mgr.RecoveryAction{
			Type:  mgr.ServiceRestart,
			Delay: 5000 * time.Millisecond,
		},
		mgr.RecoveryAction{
			Type:  mgr.ServiceRestart,
			Delay: 60000 * time.Millisecond,
		},
	}
	// set reset period as a day
	s.SetRecoveryActions(r, uint32(86400))

	return installAction + " completed.", nil
}

// Remove the service
func (windows *windowsRecord) Remove() (string, error) {
	removeAction := "Removing " + windows.description + ":"

	m, err := mgr.Connect()
	if err != nil {
		return removeAction + failed, getWindowsError(err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(windows.name)
	if err != nil {
		return removeAction + failed, getWindowsError(err)
	}
	defer s.Close()
	err = s.Delete()
	if err != nil {
		return removeAction + failed, getWindowsError(err)
	}

	return removeAction + " completed.", nil
}

// Start the service
func (windows *windowsRecord) Start() (string, error) {
	startAction := "Starting " + windows.description + ":"

	m, err := mgr.Connect()
	if err != nil {
		return startAction + failed, getWindowsError(err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(windows.name)
	if err != nil {
		return startAction + failed, getWindowsError(err)
	}
	defer s.Close()
	if err = s.Start(); err != nil {
		return startAction + failed, getWindowsError(err)
	}

	return startAction + " completed.", nil
}

// Stop the service
func (windows *windowsRecord) Stop() (string, error) {
	stopAction := "Stopping " + windows.description + ":"

	m, err := mgr.Connect()
	if err != nil {
		return stopAction + failed, getWindowsError(err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(windows.name)
	if err != nil {
		return stopAction + failed, getWindowsError(err)
	}
	defer s.Close()
	if err := stopAndWait(s); err != nil {
		return stopAction + failed, getWindowsError(err)
	}

	return stopAction + " completed.", nil
}

func stopAndWait(s *mgr.Service) error {
	// First stop the service. Then wait for the service to
	// actually stop before starting it.
	status, err := s.Control(svc.Stop)
	if err != nil {
		return err
	}

	timeDuration := time.Millisecond * 50

	timeout := time.After(getStopTimeout() + (timeDuration * 2))
	tick := time.NewTicker(timeDuration)
	defer tick.Stop()

	for status.State != svc.Stopped {
		select {
		case <-tick.C:
			status, err = s.Query()
			if err != nil {
				return err
			}
		case <-timeout:
			break
		}
	}
	return nil
}

func getStopTimeout() time.Duration {
	// For default and paths see https://support.microsoft.com/en-us/kb/146092
	defaultTimeout := time.Millisecond * 20000
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control`, registry.READ)
	if err != nil {
		return defaultTimeout
	}
	sv, _, err := key.GetStringValue("WaitToKillServiceTimeout")
	if err != nil {
		return defaultTimeout
	}
	v, err := strconv.Atoi(sv)
	if err != nil {
		return defaultTimeout
	}
	return time.Millisecond * time.Duration(v)
}

// Status - Get service status
func (windows *windowsRecord) Status() (string, error) {
	m, err := mgr.Connect()
	if err != nil {
		return "Getting status:" + failed, getWindowsError(err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(windows.name)
	if err != nil {
		return "Getting status:" + failed, getWindowsError(err)
	}
	defer s.Close()
	status, err := s.Query()
	if err != nil {
		return "Getting status:" + failed, getWindowsError(err)
	}

	return "Status: " + getWindowsServiceStateFromUint32(status.State), nil
}

// Get executable path
func execPath() (string, error) {
	var n uint32
	b := make([]uint16, syscall.MAX_PATH)
	size := uint32(len(b))

	r0, _, e1 := syscall.MustLoadDLL(
		"kernel32.dll",
	).MustFindProc(
		"GetModuleFileNameW",
	).Call(0, uintptr(unsafe.Pointer(&b[0])), uintptr(size))
	n = uint32(r0)
	if n == 0 {
		return "", e1
	}
	return string(utf16.Decode(b[0:n])), nil
}

// Get windows error
func getWindowsError(inputError error) error {
	if exiterr, ok := inputError.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			if sysErr, ok := WinErrCode[status.ExitStatus()]; ok {
				return errors.New(fmt.Sprintf("\n %s: %s \n %s", sysErr.Title, sysErr.Description, sysErr.Action))
			}
		}
	}

	return inputError
}

// Get windows service state
func getWindowsServiceStateFromUint32(state svc.State) string {
	switch state {
	case svc.Stopped:
		return "SERVICE_STOPPED"
	case svc.StartPending:
		return "SERVICE_START_PENDING"
	case svc.StopPending:
		return "SERVICE_STOP_PENDING"
	case svc.Running:
		return "SERVICE_RUNNING"
	case svc.ContinuePending:
		return "SERVICE_CONTINUE_PENDING"
	case svc.PausePending:
		return "SERVICE_PAUSE_PENDING"
	case svc.Paused:
		return "SERVICE_PAUSED"
	}
	return "SERVICE_UNKNOWN"
}

type serviceHandler struct {
	executable Executable
}

func (sh *serviceHandler) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}

	fasttick := time.Tick(500 * time.Millisecond)
	slowtick := time.Tick(2 * time.Second)
	tick := fasttick

	sh.executable.Start()
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		select {
		case <-tick:
			break
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				sh.executable.Stop()
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				tick = slowtick
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				tick = fasttick
			default:
				continue loop
			}
		}
	}
	return
}

func (windows *windowsRecord) Run(e Executable) (string, error) {
	runAction := "Running " + windows.description + ":"

	interactive, err := svc.IsAnInteractiveSession()
	if err != nil {
		return runAction + failed, getWindowsError(err)
	}
	if !interactive {
		// service called from windows service manager
		// use API provided by golang.org/x/sys/windows
		err = svc.Run(windows.name, &serviceHandler{
			executable: e,
		})
		if err != nil {
			return runAction + failed, getWindowsError(err)
		}
	} else {
		// otherwise, service should be called from terminal session
		e.Run()
	}

	return runAction + " completed.", nil
}

// GetTemplate - gets service config template
func (linux *windowsRecord) GetTemplate() string {
	return ""
}

// SetTemplate - sets service config template
func (linux *windowsRecord) SetTemplate(tplStr string) error {
	return errors.New(fmt.Sprintf("templating is not supported for windows"))
}
