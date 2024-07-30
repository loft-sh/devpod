// Copyright (c) 2023 Tailscale Inc & AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package internal

import (
	"golang.org/x/sys/windows"
)

type HGLOBAL windows.Handle
