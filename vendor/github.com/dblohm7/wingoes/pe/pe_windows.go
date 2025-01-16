// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package pe

import (
	"bytes"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func (pei *peModule) Close() error {
	return windows.FreeLibrary(windows.Handle(pei.modLock))
}

// NewPEFromBaseAddressAndSize parses the headers in a PE binary loaded
// into the current process's address space at address baseAddr with known
// size. If you do not have the size, use NewPEFromBaseAddress instead.
// Upon success it returns a non-nil *PEHeaders, otherwise it returns a nil
// *PEHeaders and a non-nil error.
func NewPEFromBaseAddressAndSize(baseAddr uintptr, size uint32) (*PEHeaders, error) {
	// Grab a strong reference to the module until we're done with it.
	var modLock windows.Handle
	if err := windows.GetModuleHandleEx(
		windows.GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS,
		(*uint16)(unsafe.Pointer(baseAddr)),
		&modLock,
	); err != nil {
		return nil, err
	}

	slc := unsafe.Slice((*byte)(unsafe.Pointer(baseAddr)), size)
	r := bytes.NewReader(slc)
	peMod := &peModule{
		Reader: r,
		peBounds: peBounds{
			base:  baseAddr,
			limit: baseAddr + uintptr(size),
		},
		modLock: uintptr(modLock),
	}

	peh, err := loadHeaders(peMod)
	if err != nil {
		peMod.Close()
		return nil, err
	}

	return peh, nil
}

// NewPEFromBaseAddress parses the headers in a PE binary loaded into the
// current process's address space at address baseAddr.
// Upon success it returns a non-nil *PEHeaders, otherwise it returns a nil
// *PEHeaders and a non-nil error.
func NewPEFromBaseAddress(baseAddr uintptr) (*PEHeaders, error) {
	var modInfo windows.ModuleInfo
	if err := windows.GetModuleInformation(
		windows.CurrentProcess(),
		windows.Handle(baseAddr),
		&modInfo,
		uint32(unsafe.Sizeof(modInfo)),
	); err != nil {
		return nil, fmt.Errorf("querying module handle: %w", err)
	}

	return NewPEFromBaseAddressAndSize(baseAddr, modInfo.SizeOfImage)
}

// NewPEFromHMODULE parses the headers in a PE binary identified by hmodule that
// is currently loaded into the current process's address space.
// Upon success it returns a non-nil *PEHeaders, otherwise it returns a nil
// *PEHeaders and a non-nil error.
func NewPEFromHMODULE(hmodule windows.Handle) (*PEHeaders, error) {
	// HMODULEs are just a loaded module's base address with the lowest two
	// bits used for flags (see docs for LoadLibraryExW).
	return NewPEFromBaseAddress(uintptr(hmodule) & ^uintptr(3))
}

// NewPEFromDLL parses the headers in a PE binary identified by dll that
// is currently loaded into the current process's address space.
// Upon success it returns a non-nil *PEHeaders, otherwise it returns a nil
// *PEHeaders and a non-nil error.
func NewPEFromDLL(dll *windows.DLL) (*PEHeaders, error) {
	if dll == nil || dll.Handle == 0 {
		return nil, os.ErrInvalid
	}

	return NewPEFromHMODULE(dll.Handle)
}

// NewPEFromLazyDLL parses the headers in a PE binary identified by ldll that
// is currently loaded into the current process's address space.
// Upon success it returns a non-nil *PEHeaders, otherwise it returns a nil
// *PEHeaders and a non-nil error.
func NewPEFromLazyDLL(ldll *windows.LazyDLL) (*PEHeaders, error) {
	if ldll == nil {
		return nil, os.ErrInvalid
	}
	if err := ldll.Load(); err != nil {
		return nil, err
	}

	return NewPEFromHMODULE(windows.Handle(ldll.Handle()))
}

// NewPEFromFileHandle parses the PE headers from hfile, an open Win32 file handle.
// It does *not* consume hfile.
// Upon success it returns a non-nil *PEHeaders, otherwise it returns a
// nil *PEHeaders and a non-nil error.
// Call Close() on the returned *PEHeaders when it is no longer needed.
func NewPEFromFileHandle(hfile windows.Handle) (*PEHeaders, error) {
	if hfile == 0 || hfile == windows.InvalidHandle {
		return nil, os.ErrInvalid
	}

	// Duplicate hfile so that we don't consume it.
	var hfileDup windows.Handle
	cp := windows.CurrentProcess()
	if err := windows.DuplicateHandle(
		cp,
		hfile,
		cp,
		&hfileDup,
		0,
		false,
		windows.DUPLICATE_SAME_ACCESS,
	); err != nil {
		return nil, err
	}

	return newPEFromFile(os.NewFile(uintptr(hfileDup), "PEFromFileHandle"))
}

func checkMachine(r peReader, machine uint16) bool {
	// In-memory modules should always have a machine type that matches our own.
	// (okay, so that's kinda sorta untrue with respect to WOW64, but that's
	// a _very_ obscure use case).
	_, isModule := r.(*peModule)
	return !isModule || machine == expectedMachineForGOARCH
}
