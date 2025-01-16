// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

//go:build windows

package pe

import (
	dpe "debug/pe"
)

type optionalHeaderForGOARCH = optionalHeader64

const (
	expectedMachineForGOARCH = dpe.IMAGE_FILE_MACHINE_ARM64
)
