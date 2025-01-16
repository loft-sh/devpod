// Copyright (c) 2022 Tailscale Inc & AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package com

import (
	"syscall"
	"unsafe"

	"github.com/dblohm7/wingoes"
)

// IUnknown is the base COM interface.
type IUnknown interface {
	QueryInterface(iid *IID) (IUnknown, error)
	AddRef() int32
	Release() int32
}

// This is a sentinel that indicates that a struct implements the COM ABI.
// Only IUnknownABI should implement this.
type hasVTable interface {
	vtable() *uintptr
}

// IUnknownABI describes the ABI of the IUnknown interface (ie, a vtable).
type IUnknownABI struct {
	Vtbl *uintptr
}

func (abi IUnknownABI) vtable() *uintptr {
	return abi.Vtbl
}

// ABI is a type constraint allowing the COM ABI, or any struct that embeds it.
type ABI interface {
	hasVTable
}

// PUnknown is a type constraint for types that both implement IUnknown and
// are also pointers to a COM ABI.
type PUnknown[A ABI] interface {
	IUnknown
	*A
}

// ABIReceiver is the type that receives COM interface pointers from COM
// method outparams.
type ABIReceiver **IUnknownABI

// NewABIReceiver instantiates a new ABIReceiver.
func NewABIReceiver() ABIReceiver {
	return ABIReceiver(new(*IUnknownABI))
}

// ReleaseABI releases a COM object. Finalizers must always invoke this function
// when destroying COM interfaces.
func ReleaseABI(p **IUnknownABI) {
	(*p).Release()
}

// QueryInterface implements the QueryInterface call for a COM interface pointer.
// iid is the desired interface ID.
func (abi *IUnknownABI) QueryInterface(iid *IID) (IUnknown, error) {
	var punk *IUnknownABI

	r, _, _ := syscall.Syscall(
		*(abi.Vtbl),
		3,
		uintptr(unsafe.Pointer(abi)),
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&punk)),
	)
	if e := wingoes.ErrorFromHRESULT(wingoes.HRESULT(r)); e.Failed() {
		return nil, e
	}

	return punk, nil
}

// AddRef implements the AddRef call for a COM interface pointer.
func (abi *IUnknownABI) AddRef() int32 {
	method := unsafe.Slice(abi.Vtbl, 3)[1]

	r, _, _ := syscall.Syscall(
		method,
		1,
		uintptr(unsafe.Pointer(abi)),
		0,
		0,
	)

	return int32(r)
}

// Release implements the Release call for a COM interface pointer.
func (abi *IUnknownABI) Release() int32 {
	method := unsafe.Slice(abi.Vtbl, 3)[2]

	r, _, _ := syscall.Syscall(
		method,
		1,
		uintptr(unsafe.Pointer(abi)),
		0,
		0,
	)

	return int32(r)
}
