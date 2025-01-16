// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

//go:build windows

package automation

//go:generate go run golang.org/x/sys/windows/mkwinsyscall -output zsyscall_windows.go mksyscall.go
//go:generate go run golang.org/x/tools/cmd/goimports -w zsyscall_windows.go

//sys sysAllocString(str *uint16) (ret BSTR) = oleaut32.SysAllocString
//sys sysAllocStringLen(str *uint16, strLen uint32) (ret BSTR) = oleaut32.SysAllocStringLen
//sys sysFreeString(bstr BSTR) = oleaut32.SysFreeString
//sys sysStringLen(bstr BSTR) (ret uint32) = oleaut32.SysStringLen
