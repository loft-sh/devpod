// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

//go:build go1.21 && !plan9

package cli

import "syscall"

func init() {
	sigPipe = syscall.SIGPIPE
}
