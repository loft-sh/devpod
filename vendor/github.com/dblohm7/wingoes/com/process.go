// Copyright (c) 2022 Tailscale Inc & AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package com

import (
	"os"
	"runtime"

	"github.com/dblohm7/wingoes"
	"golang.org/x/sys/windows"
)

// ProcessType is an enumeration that specifies the type of the current process
// when calling StartRuntime.
type ProcessType uint

const (
	// ConsoleApp is a text-mode Windows program.
	ConsoleApp = ProcessType(iota)
	// Service is a Windows service.
	Service
	// GUIApp is a GUI-mode Windows program.
	GUIApp

	// Note: Even though this implementation is not yet internally distinguishing
	// between console apps and services, this distinction may be useful in the
	// future. For example, a service could receive more restrictive default
	// security settings than a console app.
	// Having this as part of the API now avoids future breakage.
)

// StartRuntime permanently initializes COM for the remaining lifetime of the
// current process. To avoid errors, it should be called as early as possible
// during program initialization. When processType == GUIApp, the current
// OS thread becomes permanently locked to the current goroutine; any subsequent
// GUI *must* be created on the same OS thread.
// An excellent location to call StartRuntime is in the init function of the
// main package.
func StartRuntime(processType ProcessType) error {
	return StartRuntimeWithDACL(processType, nil)
}

// StartRuntimeWithDACL permanently initializes COM for the remaining lifetime
// of the current process. To avoid errors, it should be called as early as
// possible during program initialization. When processType == GUIApp, the
// current OS thread becomes permanently locked to the current goroutine; any
// subsequent GUI *must* be created on the same OS thread. dacl is an ACL that
// controls access of other processes connecting to the current process over COM.
// For further information about COM access control, look up the COM_RIGHTS_*
// access flags in the Windows developer documentation.
// An excellent location to call StartRuntimeWithDACL is in the init function of
// the main package.
func StartRuntimeWithDACL(processType ProcessType, dacl *windows.ACL) error {
	runtime.LockOSThread()

	defer func() {
		// When initializing for non-GUI processes, the OS thread may be unlocked
		// upon return from this function.
		if processType != GUIApp {
			runtime.UnlockOSThread()
		}
	}()

	switch processType {
	case ConsoleApp, Service:
		// Just start the MTA implicitly.
		if err := startMTAImplicitly(); err != nil {
			return err
		}
	case GUIApp:
		// For GUIApp, we want the current OS thread to enter a single-threaded
		// apartment (STA). However, we want all other OS threads to reside inside
		// a multi-threaded apartment (MTA). The way to so this is to first start
		// the MTA implicitly, affecting all OS threads who have not yet explicitly
		// entered a COM apartment...
		if err := startMTAImplicitly(); err != nil {
			runtime.UnlockOSThread()
			return err
		}
		// ...and then subsequently explicitly enter a STA on this OS thread, which
		// automatically removes this OS thread from the MTA.
		if err := enterSTA(); err != nil {
			runtime.UnlockOSThread()
			return err
		}
		// From this point forward, we must never unlock the OS thread.
	default:
		return os.ErrInvalid
	}

	// Order is extremely important here: initSecurity must be called immediately
	// after apartments are set up, but before doing anything else.
	if err := initSecurity(dacl); err != nil {
		return err
	}

	// By default, for compatibility reasons, COM internally sets a catch-all
	// exception handler at its API boundary. This is dangerous, so we override it.
	// This work must happen after security settings are initialized, but before
	// anything "significant" is done with COM.
	globalOpts, err := CreateInstance[GlobalOptions](CLSID_GlobalOptions)
	if err != nil {
		return err
	}

	err = globalOpts.Set(COMGLB_EXCEPTION_HANDLING, COMGLB_EXCEPTION_DONOT_HANDLE_ANY)

	// The BSTR cache never invalidates itself, so we disable it unconditionally.
	// We do this here to ensure that the BSTR cache is off before anything
	// can possibly start using oleaut32.dll.
	setOaNoCache()

	return err
}

// startMTAImplicitly creates an implicit multi-threaded apartment (MTA) for
// all threads in a process that do not otherwise explicitly enter a COM apartment.
func startMTAImplicitly() error {
	// CoIncrementMTAUsage is the modern API to use for creating the MTA implicitly,
	// however we may fall back to a legacy mechanism when the former API is unavailable.
	if err := procCoIncrementMTAUsage.Find(); err != nil {
		return startMTAImplicitlyLegacy()
	}

	// We do not retain cookie beyond this function, as we have no intention of
	// tearing any of this back down.
	var cookie coMTAUsageCookie
	hr := coIncrementMTAUsage(&cookie)
	if e := wingoes.ErrorFromHRESULT(hr); e.Failed() {
		return e
	}

	return nil
}

// startMTAImplicitlyLegacy works by having a background OS thread explicitly enter
// the multi-threaded apartment. All other OS threads that have not explicitly
// entered an apartment will become implicit members of that MTA. This function is
// written assuming that the current OS thread has already been locked.
func startMTAImplicitlyLegacy() error {
	// We need to start the MTA on a background OS thread, HOWEVER we also want this
	// to happen synchronously, so we wait on c for MTA initialization to complete.
	c := make(chan error)
	go bgMTASustainer(c)
	return <-c
}

// bgMTASustainer locks the current goroutine to the current OS thread, enters
// the COM multi-threaded apartment, and then blocks for the remainder of the
// process's lifetime. It sends its result to c so that startMTAImplicitlyLegacy
// can wait for the MTA to be ready before proceeding.
func bgMTASustainer(c chan error) {
	runtime.LockOSThread()
	err := enterMTA()
	c <- err
	if err != nil {
		// We didn't enter the MTA, so just unlock and bail.
		runtime.UnlockOSThread()
		return
	}
	select {}
}

// enterMTA causes the current OS thread to explicitly declare itself to be a
// member of COM's multi-threaded apartment. Note that this function affects
// thread-local state, so use carefully!
func enterMTA() error {
	return coInit(windows.COINIT_MULTITHREADED)
}

// enterSTA causes the current OS thread to create and enter a single-threaded
// apartment. The current OS thread must be locked and remain locked for the
// duration of the thread's time in the apartment. For our purposes, the calling
// OS thread never leaves the STA, so it must effectively remain locked for
// the remaining lifetime of the process. A single-threaded apartment should be
// used if and only if an OS thread is going to be creating windows and pumping
// messages; STAs are NOT generic containers for single-threaded COM code,
// contrary to popular belief. Note that this function affects thread-local
// state, so use carefully!
func enterSTA() error {
	return coInit(windows.COINIT_APARTMENTTHREADED)
}

// coInit is a wrapper for CoInitializeEx that properly handles the S_FALSE
// error code (x/sys/windows.CoInitializeEx does not).
func coInit(apartment uint32) error {
	hr := coInitializeEx(0, apartment)
	if e := wingoes.ErrorFromHRESULT(hr); e.Failed() {
		return e
	}

	return nil
}

const (
	authSvcCOMChooses = -1
)

// initSecurity initializes COM security using the ACL specified by dacl.
// A nil dacl implies that a default ACL should be used instead.
func initSecurity(dacl *windows.ACL) error {
	sd, err := buildSecurityDescriptor(dacl)
	if err != nil {
		return err
	}

	caps := authCapNone
	if sd == nil {
		// For COM to fall back to system-wide defaults, we need to set this bit.
		caps |= authCapAppID
	}

	hr := coInitializeSecurity(
		sd,
		authSvcCOMChooses,
		nil, // authSvc (not used because previous arg is authSvcCOMChooses)
		0,   // Reserved, must be 0
		rpcAuthnLevelDefault,
		rpcImpLevelIdentify,
		nil, // authlist: use defaults
		caps,
		0, // Reserved, must be 0
	)
	if e := wingoes.ErrorFromHRESULT(hr); e.Failed() {
		return e
	}

	return nil
}

// buildSecurityDescriptor inserts dacl into a valid security descriptor for use
// with CoInitializeSecurity. A nil dacl results in a nil security descriptor,
// which we consider to be a valid "use defaults" sentinel.
func buildSecurityDescriptor(dacl *windows.ACL) (*windows.SECURITY_DESCRIPTOR, error) {
	if dacl == nil {
		// Not an error, just use defaults.
		return nil, nil
	}

	sd, err := windows.NewSecurityDescriptor()
	if err != nil {
		return nil, err
	}

	if err := sd.SetDACL(dacl, true, false); err != nil {
		return nil, err
	}

	// CoInitializeSecurity will fail unless the SD's owner and group are both set.
	userSIDs, err := wingoes.CurrentProcessUserSIDs()
	if err != nil {
		return nil, err
	}

	if err := sd.SetOwner(userSIDs.User, false); err != nil {
		return nil, err
	}

	if err := sd.SetGroup(userSIDs.PrimaryGroup, false); err != nil {
		return nil, err
	}

	return sd, nil
}
