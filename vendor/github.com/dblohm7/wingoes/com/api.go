// Copyright (c) 2022 Tailscale Inc & AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package com

import (
	"unsafe"

	"github.com/dblohm7/wingoes"
)

// MustGetAppID parses s, a string containing an app ID and returns a pointer to the
// parsed AppID. s must be specified in the format "{XXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}".
// If there is an error parsing s, MustGetAppID panics.
func MustGetAppID(s string) *AppID {
	return (*AppID)(unsafe.Pointer(wingoes.MustGetGUID(s)))
}

// MustGetCLSID parses s, a string containing a CLSID and returns a pointer to the
// parsed CLSID. s must be specified in the format "{XXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}".
// If there is an error parsing s, MustGetCLSID panics.
func MustGetCLSID(s string) *CLSID {
	return (*CLSID)(unsafe.Pointer(wingoes.MustGetGUID(s)))
}

// MustGetIID parses s, a string containing an IID and returns a pointer to the
// parsed IID. s must be specified in the format "{XXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}".
// If there is an error parsing s, MustGetIID panics.
func MustGetIID(s string) *IID {
	return (*IID)(unsafe.Pointer(wingoes.MustGetGUID(s)))
}

// MustGetServiceID parses s, a string containing a service ID and returns a pointer to the
// parsed ServiceID. s must be specified in the format "{XXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}".
// If there is an error parsing s, MustGetServiceID panics.
func MustGetServiceID(s string) *ServiceID {
	return (*ServiceID)(unsafe.Pointer(wingoes.MustGetGUID(s)))
}

func getCurrentApartmentInfo() (aptInfo, error) {
	var info aptInfo
	hr := coGetApartmentType(&info.apt, &info.qualifier)
	if err := wingoes.ErrorFromHRESULT(hr); err.Failed() {
		return info, err
	}

	return info, nil
}

// aptChecker is a function that applies an arbitrary predicate to an OS thread's
// apartment information, returning true if the input satisifes that predicate.
type aptChecker func(*aptInfo) bool

// checkCurrentApartment obtains information about the COM apartment that the
// current OS thread resides in, and then passes that information to chk,
// which evaluates that information and determines the return value.
func checkCurrentApartment(chk aptChecker) bool {
	info, err := getCurrentApartmentInfo()
	if err != nil {
		return false
	}

	return chk(&info)
}

// AssertCurrentOSThreadSTA checks if the current OS thread resides in a
// single-threaded apartment, and if not, panics.
func AssertCurrentOSThreadSTA() {
	if IsCurrentOSThreadSTA() {
		return
	}
	panic("current OS thread does not reside in a single-threaded apartment")
}

// IsCurrentOSThreadSTA checks if the current OS thread resides in a
// single-threaded apartment and returns true if so.
func IsCurrentOSThreadSTA() bool {
	chk := func(i *aptInfo) bool {
		return i.apt == coAPTTYPE_STA || i.apt == coAPTTYPE_MAINSTA
	}

	return checkCurrentApartment(chk)
}

// AssertCurrentOSThreadMTA checks if the current OS thread resides in the
// multi-threaded apartment, and if not, panics.
func AssertCurrentOSThreadMTA() {
	if IsCurrentOSThreadMTA() {
		return
	}
	panic("current OS thread does not reside in the multi-threaded apartment")
}

// IsCurrentOSThreadMTA checks if the current OS thread resides in the
// multi-threaded apartment and returns true if so.
func IsCurrentOSThreadMTA() bool {
	chk := func(i *aptInfo) bool {
		return i.apt == coAPTTYPE_MTA
	}

	return checkCurrentApartment(chk)
}

// createInstanceWithCLSCTX creates a new garbage-collected COM object of type T
// using class clsid. clsctx determines the acceptable location for hosting the
// COM object (in-process, local but out-of-process, or remote).
func createInstanceWithCLSCTX[T Object](clsid *CLSID, clsctx coCLSCTX) (T, error) {
	var t T

	iid := t.IID()
	ppunk := NewABIReceiver()

	hr := coCreateInstance(
		clsid,
		nil,
		clsctx,
		iid,
		ppunk,
	)
	if err := wingoes.ErrorFromHRESULT(hr); err.Failed() {
		return t, err
	}

	return t.Make(ppunk).(T), nil
}

// CreateInstance instantiates a new in-process COM object of type T
// using class clsid.
func CreateInstance[T Object](clsid *CLSID) (T, error) {
	return createInstanceWithCLSCTX[T](clsid, coCLSCTX_INPROC_SERVER)
}

// CreateInstance instantiates a new local, out-of-process COM object of type T
// using class clsid.
func CreateOutOfProcessInstance[T Object](clsid *CLSID) (T, error) {
	return createInstanceWithCLSCTX[T](clsid, coCLSCTX_LOCAL_SERVER)
}
