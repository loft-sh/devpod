// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

//go:build !windows

package pe

func (pei *peModule) Close() error {
	return nil
}

func checkMachine(pe peReader, machine uint16) bool {
	return true
}
