//go:build windows || wasm || plan9 || tamago

// SPDX-License-Identifier: MIT

package rwcancel

type RWCancel struct{}

func (*RWCancel) Cancel() {}
