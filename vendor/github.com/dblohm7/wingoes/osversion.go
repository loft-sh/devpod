// Copyright (c) 2022 Tailscale Inc & AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package wingoes

import (
	"fmt"
	"sync"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	verOnce sync.Once
	verInfo osVersionInfo // must access via getVersionInfo()
)

// osVersionInfo is more compact than windows.OsVersionInfoEx, which contains
// extraneous information.
type osVersionInfo struct {
	major       uint32
	minor       uint32
	build       uint32
	servicePack uint16
	str         string
	isDC        bool
	isServer    bool
}

const (
	_VER_NT_WORKSTATION       = 1
	_VER_NT_DOMAIN_CONTROLLER = 2
	_VER_NT_SERVER            = 3
)

func getVersionInfo() *osVersionInfo {
	verOnce.Do(func() {
		osv := windows.RtlGetVersion()
		verInfo = osVersionInfo{
			major:       osv.MajorVersion,
			minor:       osv.MinorVersion,
			build:       osv.BuildNumber,
			servicePack: osv.ServicePackMajor,
			str:         fmt.Sprintf("%d.%d.%d", osv.MajorVersion, osv.MinorVersion, osv.BuildNumber),
			isDC:        osv.ProductType == _VER_NT_DOMAIN_CONTROLLER,
			// Domain Controllers are also implicitly servers.
			isServer: osv.ProductType == _VER_NT_DOMAIN_CONTROLLER || osv.ProductType == _VER_NT_SERVER,
		}
		// UBR is only available on Windows 10 and 11 (MajorVersion == 10).
		if osv.MajorVersion == 10 {
			if ubr, err := getUBR(); err == nil {
				verInfo.str = fmt.Sprintf("%s.%d", verInfo.str, ubr)
			}
		}
	})
	return &verInfo
}

// getUBR returns the "update build revision," ie. the fourth component of the
// version string found on Windows 10 and Windows 11 systems.
func getUBR() (uint32, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return 0, err
	}
	defer key.Close()

	val, valType, err := key.GetIntegerValue("UBR")
	if err != nil {
		return 0, err
	}
	if valType != registry.DWORD {
		return 0, registry.ErrUnexpectedType
	}

	return uint32(val), nil
}

// GetOSVersionString returns the Windows version of the current machine in
// dotted-decimal form. The version string contains 3 components on Windows 7
// and 8.x, and 4 components on Windows 10 and 11.
func GetOSVersionString() string {
	return getVersionInfo().String()
}

// IsWinServer returns true if and only if this computer's version of Windows is
// a server edition.
func IsWinServer() bool {
	return getVersionInfo().isServer
}

// IsWinDomainController returs true if this computer's version of Windows is
// configured to act as a domain controller.
func IsWinDomainController() bool {
	return getVersionInfo().isDC
}

// IsWin7SP1OrGreater returns true when running on Windows 7 SP1 or newer.
func IsWin7SP1OrGreater() bool {
	if IsWin8OrGreater() {
		return true
	}

	vi := getVersionInfo()
	return vi.major == 6 && vi.minor == 1 && vi.servicePack > 0
}

// IsWin8OrGreater returns true when running on Windows 8.0 or newer.
func IsWin8OrGreater() bool {
	return getVersionInfo().isVersionOrGreater(6, 2, 0)
}

// IsWin8Point1OrGreater returns true when running on Windows 8.1 or newer.
func IsWin8Point1OrGreater() bool {
	return getVersionInfo().isVersionOrGreater(6, 3, 0)
}

// IsWin10OrGreater returns true when running on any build of Windows 10 or newer.
func IsWin10OrGreater() bool {
	return getVersionInfo().major >= 10
}

// Win10BuildConstant encodes build numbers for the various editions of Windows 10,
// for use with IsWin10BuildOrGreater.
type Win10BuildConstant uint32

const (
	Win10BuildNov2015      = Win10BuildConstant(10586)
	Win10BuildAnniversary  = Win10BuildConstant(14393)
	Win10BuildCreators     = Win10BuildConstant(15063)
	Win10BuildFallCreators = Win10BuildConstant(16299)
	Win10BuildApr2018      = Win10BuildConstant(17134)
	Win10BuildSep2018      = Win10BuildConstant(17763)
	Win10BuildMay2019      = Win10BuildConstant(18362)
	Win10BuildSep2019      = Win10BuildConstant(18363)
	Win10BuildApr2020      = Win10BuildConstant(19041)
	Win10Build20H2         = Win10BuildConstant(19042)
	Win10Build21H1         = Win10BuildConstant(19043)
	Win10Build21H2         = Win10BuildConstant(19044)
)

// IsWin10BuildOrGreater returns true when running on the specified Windows 10
// build, or newer.
func IsWin10BuildOrGreater(build Win10BuildConstant) bool {
	return getVersionInfo().isWin10BuildOrGreater(uint32(build))
}

// Win11BuildConstant encodes build numbers for the various editions of Windows 11,
// for use with IsWin11BuildOrGreater.
type Win11BuildConstant uint32

const (
	Win11BuildRTM  = Win11BuildConstant(22000)
	Win11Build22H2 = Win11BuildConstant(22621)
)

// IsWin11OrGreater returns true when running on any release of Windows 11,
// or newer.
func IsWin11OrGreater() bool {
	return IsWin11BuildOrGreater(Win11BuildRTM)
}

// IsWin11BuildOrGreater returns true when running on the specified Windows 11
// build, or newer.
func IsWin11BuildOrGreater(build Win11BuildConstant) bool {
	// Under the hood, Windows 11 is just Windows 10 with a sufficiently advanced
	// build number.
	return getVersionInfo().isWin10BuildOrGreater(uint32(build))
}

func (osv *osVersionInfo) String() string {
	return osv.str
}

func (osv *osVersionInfo) isWin10BuildOrGreater(build uint32) bool {
	return osv.isVersionOrGreater(10, 0, build)
}

func (osv *osVersionInfo) isVersionOrGreater(major, minor, build uint32) bool {
	return isVerGE(osv.major, major, osv.minor, minor, osv.build, build)
}

func isVerGE(lmajor, rmajor, lminor, rminor, lbuild, rbuild uint32) bool {
	return lmajor > rmajor ||
		lmajor == rmajor &&
			(lminor > rminor ||
				lminor == rminor && lbuild >= rbuild)
}
