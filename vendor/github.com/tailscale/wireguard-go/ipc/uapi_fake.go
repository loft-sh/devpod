//go:build wasm || plan9

/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2023 WireGuard LLC. All Rights Reserved.
 */

package ipc

// Made up sentinel error codes for {js,wasip1}/wasm, and plan9.
const (
	IpcErrorIO        = 1
	IpcErrorInvalid   = 2
	IpcErrorPortInUse = 3
	IpcErrorUnknown   = 4
	IpcErrorProtocol  = 5
)
