// Copyright (c) 2022 Tailscale Inc & AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wingoes

import (
	"fmt"

	"golang.org/x/sys/windows"
)

type GUID = windows.GUID

// MustGetGUID parses s, a string containing a GUID and returns a pointer to the
// parsed GUID. s must be specified in the format "{XXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}".
// If there is an error parsing s, MustGetGUID panics.
func MustGetGUID(s string) *windows.GUID {
	guid, err := windows.GUIDFromString(s)
	if err != nil {
		panic(fmt.Sprintf("wingoes.MustGetGUID(%q) error %v", s, err))
	}
	return &guid
}
