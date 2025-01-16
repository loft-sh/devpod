// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

// Package prebuilt provides the pre-built artifacts for the web client.
package prebuilt

import (
	"embed"
	"io/fs"
)

//go:embed build
var embedded embed.FS

// FS returns a filesystem containing build artifacts for the web client.
func FS() fs.FS {
	// ignore error, since we know build directory will always exist,
	// otherwise go:embed above would fail
	sub, _ := fs.Sub(embedded, "build")
	return sub
}
