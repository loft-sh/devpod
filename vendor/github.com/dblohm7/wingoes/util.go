// Copyright (c) 2022 Tailscale Inc & AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package wingoes

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// UserSIDs contains pointers to the SIDs for a user and their primary group.
type UserSIDs struct {
	User         *windows.SID
	PrimaryGroup *windows.SID
}

// CurrentProcessUserSIDs returns a UserSIDs containing the SIDs of the user
// and primary group who own the current process.
func CurrentProcessUserSIDs() (*UserSIDs, error) {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return nil, err
	}
	defer token.Close()

	userInfo, err := token.GetTokenUser()
	if err != nil {
		return nil, err
	}

	primaryGroup, err := token.GetTokenPrimaryGroup()
	if err != nil {
		return nil, err
	}

	// We just want the SIDs, not the rest of the structs that were output.
	userSid, err := userInfo.User.Sid.Copy()
	if err != nil {
		return nil, err
	}

	primaryGroupSid, err := primaryGroup.PrimaryGroup.Copy()
	if err != nil {
		return nil, err
	}

	return &UserSIDs{User: userSid, PrimaryGroup: primaryGroupSid}, nil
}

// getTokenInfoVariableLen obtains variable-length token information. Use
// this function for information classes that output variable-length data.
func getTokenInfoVariableLen[T any](token windows.Token, infoClass uint32) (*T, error) {
	var buf []byte
	var desiredLen uint32

	err := windows.GetTokenInformation(token, infoClass, nil, 0, &desiredLen)

	for err == windows.ERROR_INSUFFICIENT_BUFFER {
		buf = make([]byte, desiredLen)
		err = windows.GetTokenInformation(token, infoClass, unsafe.SliceData(buf), desiredLen, &desiredLen)
	}

	if err != nil {
		return nil, err
	}

	return (*T)(unsafe.Pointer(unsafe.SliceData(buf))), nil
}

// getTokenInfoFixedLen obtains known fixed-length token information. Use this
// function for information classes that output enumerations, BOOLs, integers etc.
func getTokenInfoFixedLen[T any](token windows.Token, infoClass uint32) (result T, _ error) {
	var actualLen uint32
	err := windows.GetTokenInformation(token, infoClass, (*byte)(unsafe.Pointer(&result)), uint32(unsafe.Sizeof(result)), &actualLen)
	return result, err
}
