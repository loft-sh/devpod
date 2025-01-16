// Copyright (c) 2022 Tailscale Inc & AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package com

//go:generate go run golang.org/x/sys/windows/mkwinsyscall -output zsyscall_windows.go mksyscall.go
//go:generate go run golang.org/x/tools/cmd/goimports -w zsyscall_windows.go

//sys coCreateInstance(clsid *CLSID, unkOuter *IUnknownABI, clsctx coCLSCTX, iid *IID, ppv **IUnknownABI) (hr wingoes.HRESULT) = ole32.CoCreateInstance
//sys coGetApartmentType(aptType *coAPTTYPE, qual *coAPTTYPEQUALIFIER) (hr wingoes.HRESULT) = ole32.CoGetApartmentType
//sys coInitializeEx(reserved uintptr, flags uint32) (hr wingoes.HRESULT) = ole32.CoInitializeEx
//sys coInitializeSecurity(sd *windows.SECURITY_DESCRIPTOR, authSvcLen int32, authSvc *soleAuthenticationService, reserved1 uintptr, authnLevel rpcAuthnLevel, impLevel rpcImpersonationLevel, authList *soleAuthenticationList, capabilities authCapabilities, reserved2 uintptr) (hr wingoes.HRESULT) = ole32.CoInitializeSecurity

// We don't use '?' on coIncrementMTAUsage because that doesn't play nicely with HRESULTs. We manually check for its presence in process.go
//sys coIncrementMTAUsage(cookie *coMTAUsageCookie) (hr wingoes.HRESULT) = ole32.CoIncrementMTAUsage

// Technically this proc is __cdecl, but since it has 0 args this doesn't matter
//sys setOaNoCache() = oleaut32.SetOaNoCache

// For the following two functions we use IUnknownABI instead of IStreamABI because it makes the callsites cleaner.
//sys shCreateMemStream(pInit *byte, cbInit uint32) (stream *IUnknownABI) = shlwapi.SHCreateMemStream
//sys createStreamOnHGlobal(hglobal internal.HGLOBAL, deleteOnRelease bool, stream **IUnknownABI) (hr wingoes.HRESULT) = ole32.CreateStreamOnHGlobal
