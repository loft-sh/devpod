// Copyright (c) 2022 Tailscale Inc & AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package com

import (
	"runtime"
	"syscall"
	"unsafe"

	"github.com/dblohm7/wingoes"
)

var (
	CLSID_GlobalOptions = &CLSID{0x0000034B, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
)

var (
	IID_IGlobalOptions = &IID{0x0000015B, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
)

type GLOBALOPT_PROPERTIES int32

const (
	COMGLB_EXCEPTION_HANDLING     = GLOBALOPT_PROPERTIES(1)
	COMGLB_APPID                  = GLOBALOPT_PROPERTIES(2)
	COMGLB_RPC_THREADPOOL_SETTING = GLOBALOPT_PROPERTIES(3)
	COMGLB_RO_SETTINGS            = GLOBALOPT_PROPERTIES(4)
	COMGLB_UNMARSHALING_POLICY    = GLOBALOPT_PROPERTIES(5)
)

const (
	COMGLB_EXCEPTION_HANDLE             = 0
	COMGLB_EXCEPTION_DONOT_HANDLE_FATAL = 1
	COMGLB_EXCEPTION_DONOT_HANDLE       = 1
	COMGLB_EXCEPTION_DONOT_HANDLE_ANY   = 2
)

// IGlobalOptionsABI represents the COM ABI for the IGlobalOptions interface.
type IGlobalOptionsABI struct {
	IUnknownABI
}

// GlobalOptions is the COM object used for setting global configuration settings
// on the COM runtime. It must be called after COM runtime security has been
// initialized, but before anything else "significant" is done using COM.
type GlobalOptions struct {
	GenericObject[IGlobalOptionsABI]
}

func (abi *IGlobalOptionsABI) Set(prop GLOBALOPT_PROPERTIES, value uintptr) error {
	method := unsafe.Slice(abi.Vtbl, 5)[3]

	rc, _, _ := syscall.Syscall(
		method,
		3,
		uintptr(unsafe.Pointer(abi)),
		uintptr(prop),
		value,
	)
	if e := wingoes.ErrorFromHRESULT(wingoes.HRESULT(rc)); e.Failed() {
		return e
	}

	return nil
}

func (abi *IGlobalOptionsABI) Query(prop GLOBALOPT_PROPERTIES) (uintptr, error) {
	var result uintptr
	method := unsafe.Slice(abi.Vtbl, 5)[4]

	rc, _, _ := syscall.Syscall(
		method,
		3,
		uintptr(unsafe.Pointer(abi)),
		uintptr(prop),
		uintptr(unsafe.Pointer(&result)),
	)
	if e := wingoes.ErrorFromHRESULT(wingoes.HRESULT(rc)); e.Failed() {
		return 0, e
	}

	return result, nil
}

func (o GlobalOptions) IID() *IID {
	return IID_IGlobalOptions
}

func (o GlobalOptions) Make(r ABIReceiver) any {
	if r == nil {
		return GlobalOptions{}
	}

	runtime.SetFinalizer(r, ReleaseABI)

	pp := (**IGlobalOptionsABI)(unsafe.Pointer(r))
	return GlobalOptions{GenericObject[IGlobalOptionsABI]{Pp: pp}}
}

// UnsafeUnwrap returns the underlying IGlobalOptionsABI of the object. As the
// name implies, this is unsafe -- you had better know what you are doing!
func (o GlobalOptions) UnsafeUnwrap() *IGlobalOptionsABI {
	return *(o.Pp)
}

// Set sets the global property prop to value.
func (o GlobalOptions) Set(prop GLOBALOPT_PROPERTIES, value uintptr) error {
	p := *(o.Pp)
	return p.Set(prop, value)
}

// Query returns the value of global property prop.
func (o GlobalOptions) Query(prop GLOBALOPT_PROPERTIES) (uintptr, error) {
	p := *(o.Pp)
	return p.Query(prop)
}
