// Copyright (c) 2022 Tailscale Inc & AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package com

import (
	"fmt"
	"unsafe"
)

// GenericObject is a struct that wraps any interface that implements the COM ABI.
type GenericObject[A ABI] struct {
	Pp **A
}

func (o GenericObject[A]) pp() **A {
	return o.Pp
}

// Object is the interface that all garbage-collected instances of COM interfaces
// must implement.
type Object interface {
	// IID returns the interface ID for the object. This method may be called
	// on Objects containing the zero value, so its return value must not depend
	// on the value of the method's receiver.
	IID() *IID

	// Make converts r to an instance of a garbage-collected COM object. The type
	// of its return value must always match the type of the method's receiver.
	Make(r ABIReceiver) any
}

// EmbedsGenericObject is a type constraint matching any struct that embeds
// a GenericObject[A].
type EmbedsGenericObject[A ABI] interface {
	Object
	~struct{ GenericObject[A] }
	pp() **A
}

// As casts obj to an object of type O, or panics if obj cannot be converted to O.
func As[O Object, A ABI, PU PUnknown[A], E EmbedsGenericObject[A]](obj E) O {
	o, err := TryAs[O, A, PU](obj)
	if err != nil {
		panic(fmt.Sprintf("wingoes.com.As error: %v", err))
	}
	return o
}

// TryAs casts obj to an object of type O, or returns an error if obj cannot be
// converted to O.
func TryAs[O Object, A ABI, PU PUnknown[A], E EmbedsGenericObject[A]](obj E) (O, error) {
	var o O

	iid := o.IID()
	p := (PU)(unsafe.Pointer(*(obj.pp())))

	i, err := p.QueryInterface(iid)
	if err != nil {
		return o, err
	}

	r := NewABIReceiver()
	*r = i.(*IUnknownABI)

	return o.Make(r).(O), nil
}

// IsSameObject returns true when both l and r refer to the same underlying object.
func IsSameObject[AL, AR ABI, PL PUnknown[AL], PR PUnknown[AR], EL EmbedsGenericObject[AL], ER EmbedsGenericObject[AR]](l EL, r ER) bool {
	pl := (PL)(unsafe.Pointer(*(l.pp())))
	ul, err := pl.QueryInterface(IID_IUnknown)
	if err != nil {
		return false
	}
	defer ul.Release()

	pr := (PR)(unsafe.Pointer(*(r.pp())))
	ur, err := pr.QueryInterface(IID_IUnknown)
	if err != nil {
		return false
	}
	defer ur.Release()

	return ul.(*IUnknownABI) == ur.(*IUnknownABI)
}
