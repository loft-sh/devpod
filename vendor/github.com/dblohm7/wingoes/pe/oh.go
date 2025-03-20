// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package pe

import (
	dpe "debug/pe"
	"unsafe"
)

// OptionalHeader provides the fields of a PE/COFF optional header. Since the
// underlying format differs depending on whether the PE binary is 32-bit or
// 64-bit, this type provides a unified interface.
type OptionalHeader interface {
	GetMagic() uint16
	GetLinkerVersion() (major, minor uint8)
	GetSizeOfCode() uint32
	GetSizeOfInitializedData() uint32
	GetSizeOfUninitializedData() uint32
	GetAddressOfEntryPoint() uint32
	GetBaseOfCode() uint32
	GetImageBase() uint64
	GetSectionAlignment() uint32
	GetFileAlignment() uint32
	GetOperatingSystemVersion() (major, minor uint16)
	GetImageVersion() (major, minor uint16)
	GetSubsystemVersion() (major, minor uint16)
	GetWin32Version() uint32
	GetSizeOfImage() uint32
	GetSizeOfHeaders() uint32
	GetCheckSum() uint32
	GetSubsystem() uint16
	GetDllCharacteristics() uint16
	GetSizeOfStackReserve() uint64
	GetSizeOfStackCommit() uint64
	GetSizeOfHeapReserve() uint64
	GetSizeOfHeapCommit() uint64
	GetLoaderFlags() uint32
	GetDataDirectory() []DataDirectoryEntry

	SizeOf() uint16 // Size of the underlying struct, in bytes
}

type optionalHeader32 dpe.OptionalHeader32

func (oh *optionalHeader32) GetMagic() uint16 {
	return oh.Magic
}

func (oh *optionalHeader32) GetLinkerVersion() (major, minor uint8) {
	return oh.MajorLinkerVersion, oh.MinorLinkerVersion
}

func (oh *optionalHeader32) GetSizeOfCode() uint32 {
	return oh.SizeOfCode
}

func (oh *optionalHeader32) GetSizeOfInitializedData() uint32 {
	return oh.SizeOfInitializedData
}

func (oh *optionalHeader32) GetSizeOfUninitializedData() uint32 {
	return oh.SizeOfUninitializedData
}

func (oh *optionalHeader32) GetAddressOfEntryPoint() uint32 {
	return oh.AddressOfEntryPoint
}

func (oh *optionalHeader32) GetBaseOfCode() uint32 {
	return oh.BaseOfCode
}

func (oh *optionalHeader32) GetImageBase() uint64 {
	return uint64(oh.ImageBase)
}

func (oh *optionalHeader32) GetSectionAlignment() uint32 {
	return oh.SectionAlignment
}

func (oh *optionalHeader32) GetFileAlignment() uint32 {
	return oh.FileAlignment
}

func (oh *optionalHeader32) GetOperatingSystemVersion() (major, minor uint16) {
	return oh.MajorOperatingSystemVersion, oh.MinorOperatingSystemVersion
}

func (oh *optionalHeader32) GetImageVersion() (major, minor uint16) {
	return oh.MajorImageVersion, oh.MinorImageVersion
}

func (oh *optionalHeader32) GetSubsystemVersion() (major, minor uint16) {
	return oh.MajorSubsystemVersion, oh.MinorSubsystemVersion
}

func (oh *optionalHeader32) GetWin32Version() uint32 {
	return oh.Win32VersionValue
}

func (oh *optionalHeader32) GetSizeOfImage() uint32 {
	return oh.SizeOfImage
}

func (oh *optionalHeader32) GetSizeOfHeaders() uint32 {
	return oh.SizeOfHeaders
}

func (oh *optionalHeader32) GetCheckSum() uint32 {
	return oh.CheckSum
}

func (oh *optionalHeader32) GetSubsystem() uint16 {
	return oh.Subsystem
}

func (oh *optionalHeader32) GetDllCharacteristics() uint16 {
	return oh.DllCharacteristics
}

func (oh *optionalHeader32) GetSizeOfStackReserve() uint64 {
	return uint64(oh.SizeOfStackReserve)
}

func (oh *optionalHeader32) GetSizeOfStackCommit() uint64 {
	return uint64(oh.SizeOfStackCommit)
}

func (oh *optionalHeader32) GetSizeOfHeapReserve() uint64 {
	return uint64(oh.SizeOfHeapReserve)
}

func (oh *optionalHeader32) GetSizeOfHeapCommit() uint64 {
	return uint64(oh.SizeOfHeapCommit)
}

func (oh *optionalHeader32) GetLoaderFlags() uint32 {
	return oh.LoaderFlags
}

func (oh *optionalHeader32) GetDataDirectory() []DataDirectoryEntry {
	cnt := oh.NumberOfRvaAndSizes
	if maxCnt := uint32(len(oh.DataDirectory)); cnt > maxCnt {
		cnt = maxCnt
	}
	return oh.DataDirectory[:cnt]
}

func (oh *optionalHeader32) SizeOf() uint16 {
	return uint16(unsafe.Sizeof(*oh))
}

type optionalHeader64 dpe.OptionalHeader64

func (oh *optionalHeader64) GetMagic() uint16 {
	return oh.Magic
}

func (oh *optionalHeader64) GetLinkerVersion() (major, minor uint8) {
	return oh.MajorLinkerVersion, oh.MinorLinkerVersion
}

func (oh *optionalHeader64) GetSizeOfCode() uint32 {
	return oh.SizeOfCode
}

func (oh *optionalHeader64) GetSizeOfInitializedData() uint32 {
	return oh.SizeOfInitializedData
}

func (oh *optionalHeader64) GetSizeOfUninitializedData() uint32 {
	return oh.SizeOfUninitializedData
}

func (oh *optionalHeader64) GetAddressOfEntryPoint() uint32 {
	return oh.AddressOfEntryPoint
}

func (oh *optionalHeader64) GetBaseOfCode() uint32 {
	return oh.BaseOfCode
}

func (oh *optionalHeader64) GetImageBase() uint64 {
	return oh.ImageBase
}

func (oh *optionalHeader64) GetSectionAlignment() uint32 {
	return oh.SectionAlignment
}

func (oh *optionalHeader64) GetFileAlignment() uint32 {
	return oh.FileAlignment
}

func (oh *optionalHeader64) GetOperatingSystemVersion() (major, minor uint16) {
	return oh.MajorOperatingSystemVersion, oh.MinorOperatingSystemVersion
}

func (oh *optionalHeader64) GetImageVersion() (major, minor uint16) {
	return oh.MajorImageVersion, oh.MinorImageVersion
}

func (oh *optionalHeader64) GetSubsystemVersion() (major, minor uint16) {
	return oh.MajorSubsystemVersion, oh.MinorSubsystemVersion
}

func (oh *optionalHeader64) GetWin32Version() uint32 {
	return oh.Win32VersionValue
}

func (oh *optionalHeader64) GetSizeOfImage() uint32 {
	return oh.SizeOfImage
}

func (oh *optionalHeader64) GetSizeOfHeaders() uint32 {
	return oh.SizeOfHeaders
}

func (oh *optionalHeader64) GetCheckSum() uint32 {
	return oh.CheckSum
}

func (oh *optionalHeader64) GetSubsystem() uint16 {
	return oh.Subsystem
}

func (oh *optionalHeader64) GetDllCharacteristics() uint16 {
	return oh.DllCharacteristics
}

func (oh *optionalHeader64) GetSizeOfStackReserve() uint64 {
	return oh.SizeOfStackReserve
}

func (oh *optionalHeader64) GetSizeOfStackCommit() uint64 {
	return oh.SizeOfStackCommit
}

func (oh *optionalHeader64) GetSizeOfHeapReserve() uint64 {
	return oh.SizeOfHeapReserve
}

func (oh *optionalHeader64) GetSizeOfHeapCommit() uint64 {
	return oh.SizeOfHeapCommit
}

func (oh *optionalHeader64) GetLoaderFlags() uint32 {
	return oh.LoaderFlags
}

func (oh *optionalHeader64) GetDataDirectory() []DataDirectoryEntry {
	cnt := oh.NumberOfRvaAndSizes
	if maxCnt := uint32(len(oh.DataDirectory)); cnt > maxCnt {
		cnt = maxCnt
	}
	return oh.DataDirectory[:cnt]
}

func (oh *optionalHeader64) SizeOf() uint16 {
	return uint16(unsafe.Sizeof(*oh))
}
