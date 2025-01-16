// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

//go:build windows

package automation

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// BSTR is the string format used by COM Automation. They are not garbage
// collected and must be explicitly closed when no longer needed.
type BSTR uintptr

// NewBSTR creates a new BSTR from string s.
func NewBSTR(s string) BSTR {
	buf, err := windows.UTF16FromString(s)
	if err != nil {
		return 0
	}
	return NewBSTRFromUTF16(buf)
}

// NewBSTR creates a new BSTR from slice us, which contains UTF-16 code units.
func NewBSTRFromUTF16(us []uint16) BSTR {
	if len(us) == 0 {
		return 0
	}
	return sysAllocStringLen(unsafe.SliceData(us), uint32(len(us)))
}

// NewBSTR creates a new BSTR from up, a C-style string pointer to UTF-16 code units.
func NewBSTRFromUTF16Ptr(up *uint16) BSTR {
	if up == nil {
		return 0
	}
	return sysAllocString(up)
}

// Len returns the length of bs in code units.
func (bs *BSTR) Len() uint32 {
	return sysStringLen(*bs)
}

// String returns the contents of bs as a Go string.
func (bs *BSTR) String() string {
	return windows.UTF16ToString(bs.toUTF16())
}

// toUTF16 is unsafe for general use because it returns a pointer that is
// not managed by the Go GC.
func (bs *BSTR) toUTF16() []uint16 {
	return unsafe.Slice(bs.toUTF16Ptr(), bs.Len())
}

// ToUTF16 returns the contents of bs as a slice of UTF-16 code units.
func (bs *BSTR) ToUTF16() []uint16 {
	return append([]uint16{}, bs.toUTF16()...)
}

// toUTF16Ptr is unsafe for general use because it returns a pointer that is
// not managed by the Go GC.
func (bs *BSTR) toUTF16Ptr() *uint16 {
	return (*uint16)(unsafe.Pointer(*bs))
}

// ToUTF16 returns the contents of bs as C-style string pointer to UTF-16 code units.
func (bs *BSTR) ToUTF16Ptr() *uint16 {
	return unsafe.SliceData(bs.ToUTF16())
}

// Clone creates a clone of bs whose lifetime becomes independent of the original.
// It must be explicitly closed when no longer needed.
func (bs *BSTR) Clone() BSTR {
	return sysAllocStringLen(bs.toUTF16Ptr(), bs.Len())
}

// IsNil returns true if bs holds a nil value.
func (bs *BSTR) IsNil() bool {
	return *bs == 0
}

// Close frees bs.
func (bs *BSTR) Close() error {
	if *bs != 0 {
		sysFreeString(*bs)
		*bs = 0
	}
	return nil
}
