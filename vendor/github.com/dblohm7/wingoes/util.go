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

	userInfo, err := getTokenInfo[windows.Tokenuser](token, windows.TokenUser)
	if err != nil {
		return nil, err
	}

	primaryGroup, err := getTokenInfo[windows.Tokenprimarygroup](token, windows.TokenPrimaryGroup)
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

func getTokenInfo[T any](token windows.Token, infoClass uint32) (*T, error) {
	var buf []byte
	var desiredLen uint32

	err := windows.GetTokenInformation(token, infoClass, nil, 0, &desiredLen)

	for err != nil {
		if err != windows.ERROR_INSUFFICIENT_BUFFER {
			return nil, err
		}

		buf = make([]byte, desiredLen)
		err = windows.GetTokenInformation(token, infoClass, &buf[0], desiredLen, &desiredLen)
	}

	return (*T)(unsafe.Pointer(&buf[0])), nil
}
