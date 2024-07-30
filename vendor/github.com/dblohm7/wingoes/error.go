// Copyright (c) 2022 Tailscale Inc & AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package wingoes

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// Error represents various error codes that may be encountered when coding
// against Windows APIs, including HRESULTs, windows.NTStatus, and windows.Errno.
type Error HRESULT

// Errors are HRESULTs under the hood because the HRESULT encoding allows for
// all the other common types of Windows errors to be encoded within them.

const (
	hrS_OK                 = HRESULT(0)
	hrE_ABORT              = HRESULT(-((0x80004004 ^ 0xFFFFFFFF) + 1))
	hrE_FAIL               = HRESULT(-((0x80004005 ^ 0xFFFFFFFF) + 1))
	hrE_NOINTERFACE        = HRESULT(-((0x80004002 ^ 0xFFFFFFFF) + 1))
	hrE_NOTIMPL            = HRESULT(-((0x80004001 ^ 0xFFFFFFFF) + 1))
	hrE_POINTER            = HRESULT(-((0x80004003 ^ 0xFFFFFFFF) + 1))
	hrE_UNEXPECTED         = HRESULT(-((0x8000FFFF ^ 0xFFFFFFFF) + 1))
	hrTYPE_E_WRONGTYPEKIND = HRESULT(-((0x8002802A ^ 0xFFFFFFFF) + 1))
)

// S_FALSE is a peculiar HRESULT value which means that the call executed
// successfully, but returned false as its result.
const S_FALSE = HRESULT(1)

var (
	// genericError encodes an Error whose message string is very generic.
	genericError = Error(hresultFromFacilityAndCode(hrFail, facilityWin32, hrCode(windows.ERROR_UNIDENTIFIED_ERROR)))
)

// Common HRESULT codes that don't use Win32 facilities, but have meanings that
// we can manually translate to Win32 error codes.
var commonHRESULTToErrno = map[HRESULT]windows.Errno{
	hrE_ABORT:       windows.ERROR_REQUEST_ABORTED,
	hrE_FAIL:        windows.ERROR_UNIDENTIFIED_ERROR,
	hrE_NOINTERFACE: windows.ERROR_NOINTERFACE,
	hrE_NOTIMPL:     windows.ERROR_CALL_NOT_IMPLEMENTED,
	hrE_UNEXPECTED:  windows.ERROR_INTERNAL_ERROR,
}

type hrCode uint16
type hrFacility uint16
type failBit bool

const (
	hrFlagBitsMask  = 0xF8000000
	hrFacilityMax   = 0x00001FFF
	hrFacilityMask  = hrFacilityMax << 16
	hrCodeMax       = 0x0000FFFF
	hrCodeMask      = hrCodeMax
	hrFailBit       = 0x80000000
	hrCustomerBit   = 0x20000000 // Also defined as syscall.APPLICATION_ERROR
	hrFacilityNTBit = 0x10000000
)

const (
	facilityWin32 = hrFacility(7)
)

// Succeeded returns true when hr is successful, but its actual error code
// may include additional status information.
func (hr HRESULT) Succeeded() bool {
	return hr >= 0
}

// Failed returns true when hr contains a failure code.
func (hr HRESULT) Failed() bool {
	return hr < 0
}

func (hr HRESULT) isNT() bool {
	return (hr & (hrCustomerBit | hrFacilityNTBit)) == hrFacilityNTBit
}

func (hr HRESULT) isCustomer() bool {
	return (hr & hrCustomerBit) != 0
}

// isNormal returns true when the customer and NT bits are cleared, ie hr's
// encoding contains valid facility and code fields.
func (hr HRESULT) isNormal() bool {
	return (hr & (hrCustomerBit | hrFacilityNTBit)) == 0
}

// facility returns the facility bits of hr. Only valid when isNormal is true.
func (hr HRESULT) facility() hrFacility {
	return hrFacility((uint32(hr) >> 16) & hrFacilityMax)
}

// facility returns the code bits of hr. Only valid when isNormal is true.
func (hr HRESULT) code() hrCode {
	return hrCode(uint32(hr) & hrCodeMask)
}

const (
	hrFail    = failBit(true)
	hrSuccess = failBit(false)
)

func hresultFromFacilityAndCode(isFail failBit, f hrFacility, c hrCode) HRESULT {
	var r uint32
	if isFail {
		r |= hrFailBit
	}
	r |= (uint32(f) << 16) & hrFacilityMask
	r |= uint32(c) & hrCodeMask
	return HRESULT(r)
}

// ErrorFromErrno creates an Error from e.
func ErrorFromErrno(e windows.Errno) Error {
	if e == windows.ERROR_SUCCESS {
		return Error(hrS_OK)
	}
	if ue := uint32(e); (ue & hrFlagBitsMask) == hrCustomerBit {
		// syscall.APPLICATION_ERROR == hrCustomerBit, so the only other thing
		// we need to do to transform this into an HRESULT is add the fail flag
		return Error(HRESULT(ue | hrFailBit))
	}
	if uint32(e) > hrCodeMax {
		// Can't be encoded in HRESULT, return generic error instead
		return genericError
	}
	return Error(hresultFromFacilityAndCode(hrFail, facilityWin32, hrCode(e)))
}

// ErrorFromNTStatus creates an Error from s.
func ErrorFromNTStatus(s windows.NTStatus) Error {
	if s == windows.STATUS_SUCCESS {
		return Error(hrS_OK)
	}
	return Error(HRESULT(s) | hrFacilityNTBit)
}

// ErrorFromHRESULT creates an Error from hr.
func ErrorFromHRESULT(hr HRESULT) Error {
	return Error(hr)
}

// NewError converts e into an Error if e's type is supported. It returns
// both the Error and a bool indicating whether the conversion was successful.
func NewError(e any) (Error, bool) {
	switch v := e.(type) {
	case Error:
		return v, true
	case windows.NTStatus:
		return ErrorFromNTStatus(v), true
	case windows.Errno:
		return ErrorFromErrno(v), true
	case HRESULT:
		return ErrorFromHRESULT(v), true
	default:
		return ErrorFromHRESULT(hrTYPE_E_WRONGTYPEKIND), false
	}
}

// IsOK returns true when the Error is unconditionally successful.
func (e Error) IsOK() bool {
	return HRESULT(e) == hrS_OK
}

// Succeeded returns true when the Error is successful, but its error code
// may include additional status information.
func (e Error) Succeeded() bool {
	return HRESULT(e).Succeeded()
}

// Failed returns true when the Error contains a failure code.
func (e Error) Failed() bool {
	return HRESULT(e).Failed()
}

// AsHRESULT converts the Error to a HRESULT.
func (e Error) AsHRESULT() HRESULT {
	return HRESULT(e)
}

type errnoFailHandler func(hr HRESULT) windows.Errno

func (e Error) toErrno(f errnoFailHandler) windows.Errno {
	hr := HRESULT(e)

	if hr == hrS_OK {
		return windows.ERROR_SUCCESS
	}

	if hr.isCustomer() {
		return windows.Errno(uint32(e) ^ hrFailBit)
	}

	if hr.isNT() {
		return e.AsNTStatus().Errno()
	}

	if hr.facility() == facilityWin32 {
		return windows.Errno(hr.code())
	}

	if errno, ok := commonHRESULTToErrno[hr]; ok {
		return errno
	}

	return f(hr)
}

// AsError converts the Error to a windows.Errno, but panics if not possible.
func (e Error) AsErrno() windows.Errno {
	handler := func(hr HRESULT) windows.Errno {
		panic(fmt.Sprintf("wingoes.Error: Called AsErrno on a non-convertable HRESULT 0x%08X", uint32(hr)))
		return windows.ERROR_UNIDENTIFIED_ERROR
	}

	return e.toErrno(handler)
}

type ntStatusFailHandler func(hr HRESULT) windows.NTStatus

func (e Error) toNTStatus(f ntStatusFailHandler) windows.NTStatus {
	hr := HRESULT(e)

	if hr == hrS_OK {
		return windows.STATUS_SUCCESS
	}

	if hr.isNT() {
		return windows.NTStatus(hr ^ hrFacilityNTBit)
	}

	return f(hr)
}

// AsNTStatus converts the Error to a windows.NTStatus, but panics if not possible.
func (e Error) AsNTStatus() windows.NTStatus {
	handler := func(hr HRESULT) windows.NTStatus {
		panic(fmt.Sprintf("windows.Error: Called AsNTStatus on a non-NTSTATUS HRESULT 0x%08X", uint32(hr)))
		return windows.STATUS_UNSUCCESSFUL
	}

	return e.toNTStatus(handler)
}

// TryAsErrno converts the Error to a windows.Errno, or returns defval if
// such a conversion is not possible.
func (e Error) TryAsErrno(defval windows.Errno) windows.Errno {
	handler := func(hr HRESULT) windows.Errno {
		return defval
	}

	return e.toErrno(handler)
}

// TryAsNTStatus converts the Error to a windows.NTStatus, or returns defval if
// such a conversion is not possible.
func (e Error) TryAsNTStatus(defval windows.NTStatus) windows.NTStatus {
	handler := func(hr HRESULT) windows.NTStatus {
		return defval
	}

	return e.toNTStatus(handler)
}

// IsAvailableAsHRESULT returns true if e may be converted to an HRESULT.
func (e Error) IsAvailableAsHRESULT() bool {
	return true
}

// IsAvailableAsErrno returns true if e may be converted to a windows.Errno.
func (e Error) IsAvailableAsErrno() bool {
	hr := HRESULT(e)
	if hr.isCustomer() || e.IsAvailableAsNTStatus() || (hr.facility() == facilityWin32) {
		return true
	}
	_, convertable := commonHRESULTToErrno[hr]
	return convertable
}

// IsAvailableAsNTStatus returns true if e may be converted to a windows.NTStatus.
func (e Error) IsAvailableAsNTStatus() bool {
	return HRESULT(e) == hrS_OK || HRESULT(e).isNT()
}

// Error produces a human-readable message describing Error e.
func (e Error) Error() string {
	if HRESULT(e).isCustomer() {
		return windows.Errno(uint32(e) ^ hrFailBit).Error()
	}

	buf := make([]uint16, 300)
	const flags = windows.FORMAT_MESSAGE_FROM_SYSTEM | windows.FORMAT_MESSAGE_IGNORE_INSERTS
	lenExclNul, err := windows.FormatMessage(flags, 0, uint32(e), 0, buf, nil)
	if err != nil {
		return fmt.Sprintf("wingoes.Error 0x%08X", uint32(e))
	}
	for ; lenExclNul > 0 && (buf[lenExclNul-1] == '\n' || buf[lenExclNul-1] == '\r'); lenExclNul-- {
	}
	return windows.UTF16ToString(buf[:lenExclNul])
}
