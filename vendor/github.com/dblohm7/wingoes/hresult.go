// Copyright (c) 2022 Tailscale Inc & AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Note this file is explicitly available on non-Windows platforms, in order to
// aid `go generate` tooling on those platforms. It should not take a dependency
// on x/sys/windows.

package wingoes

// HRESULT is equivalent to the HRESULT type in the Win32 SDK for C/C++.
type HRESULT int32
