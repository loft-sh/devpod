// Copyright (c) 2022 Tailscale Inc & AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package com

import (
	"runtime"
)

var (
	IID_IUnknown = &IID{0x00000000, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
)

// ObjectBase is a garbage-collected instance of any COM object's base interface.
type ObjectBase struct {
	GenericObject[IUnknownABI]
}

// IID always returns IID_IUnknown.
func (o ObjectBase) IID() *IID {
	return IID_IUnknown
}

// Make produces a new instance of ObjectBase that wraps r. Its return type is
// always ObjectBase.
func (o ObjectBase) Make(r ABIReceiver) any {
	if r == nil {
		return ObjectBase{}
	}

	runtime.SetFinalizer(r, ReleaseABI)

	pp := (**IUnknownABI)(r)
	return ObjectBase{GenericObject[IUnknownABI]{Pp: pp}}
}

// UnsafeUnwrap returns the underlying IUnknownABI of the object. As the name
// implies, this is unsafe -- you had better know what you are doing!
func (o ObjectBase) UnsafeUnwrap() *IUnknownABI {
	return *(o.Pp)
}
