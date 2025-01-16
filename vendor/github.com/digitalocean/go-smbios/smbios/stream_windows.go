// Copyright 2017-2018 DigitalOcean.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package smbios

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"syscall"
	"unsafe"
)

// firmwareTableProviderSigRSMB is the identifier for the raw SMBIOS firmware table
// provider.
// It is equal to the ASCII characters 'RSMB' packed into a uint32.
// In the C++ example code in the MSDN documentation, this is specified using
// multi-byte character literals, which are automatically coerced to an integer by
// the C++ compiler.
const firmwareTableProviderSigRSMB uint32 = 0x52534d42

// smbiosDataHeaderSize is size of the "header" (non-variable) part of the
// RawSMBIOSData struct. This serves as both the offset to the actual
// SMBIOS table data, and the minimum possible size of a valid RawSMBIOSDATA
// struct (with a table length of 0).
const rawSMBIOSDataHeaderSize = 8

var (
	libKernel32 = syscall.NewLazyDLL("kernel32.dll")

	// MSDN Documentation for GetSystemFirmwareTable:
	// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724379(v=vs.85).aspx
	procGetSystemFirmwareTable = libKernel32.NewProc("GetSystemFirmwareTable")
)

// nativeEndian returns the native byte order of this system.
func nativeEndian() binary.ByteOrder {
	// Determine endianness by interpreting a uint16 as a byte slice.
	v := uint16(1)
	b := *(*[2]byte)(unsafe.Pointer(&v))

	if b[0] == 1 {
		return binary.LittleEndian
	}

	return binary.BigEndian
}

// windowsStream parses the data returned from GetSystemFirmwareTable('RSMB',...)
// and returns a stream and entrypoint that can be used to decode the system's
// SMBIOS table.
//
// When calling GetSystemFirmwareTable with FirmwareTableProviderSignature = "RSMB",
// Windows will write a RawSMBIOSData struct into the output buffer.
// Thus, `buf` is expected to contain a valid RawSMBIOSData structure.
//
//	From windows.h:
//
// 	struct RawSMBIOSData {
//		BYTE 	Used20CallingMethod;
//		BYTE	SMBIOSMajorVersion;
//		BYTE 	SMBIOSMinorVersion;
//		BYTE 	DMIRevision;
//		DWORD 	Length;
//		BYTE 	SMBIOSTableData[];
//	}
//
// Note: a DWORD is equivalent to a uint32
// See: https://msdn.microsoft.com/en-us/library/cc230318.aspx
func windowsStream(buf []byte) (io.ReadCloser, EntryPoint, error) {
	bufLen := uint32(len(buf))

	// Do an additional check to make sure the actual amount written is sane.
	if bufLen < rawSMBIOSDataHeaderSize {
		return nil, nil, fmt.Errorf("GetSystemFirmwareTable wrote less data than expected: wrote %d bytes, expected at least 8 bytes", bufLen)
	}

	tableSize := nativeEndian().Uint32(buf[4:8])
	if rawSMBIOSDataHeaderSize+tableSize > bufLen {
		return nil, nil, errors.New("reported SMBIOS table size exceeds buffer")
	}

	entryPoint := &WindowsEntryPoint{
		MajorVersion: buf[1],
		MinorVersion: buf[2],
		Revision:     buf[3],
		Size:         tableSize,
	}

	tableBuff := buf[rawSMBIOSDataHeaderSize : rawSMBIOSDataHeaderSize+tableSize]

	return ioutil.NopCloser(bytes.NewReader(tableBuff)), entryPoint, nil
}

func stream() (io.ReadCloser, EntryPoint, error) {
	// Call first with empty buffer to get size.
	r1, _, err := procGetSystemFirmwareTable.Call(
		uintptr(firmwareTableProviderSigRSMB), // FirmwareTableProviderSignature = 'RSMB'
		0,                                     // FirmwareTableID = 0
		0,                                     // pFirmwareTableBuffer = NULL
		0,                                     // BufferSize = 0
	)

	// LazyProc.Call will always return err != nil, so we need to check the primary
	// return value (r1) to determine whether or not an error occurred.
	// In this case, r1 should contain the size of the needed buffer, so it will only
	// be 0 if the function call failed for some reason.
	//
	// Godoc for LazyProc.Call:
	// https://golang.org/pkg/syscall/?GOOS=windows&GOARCH=amd64#LazyProc.Call
	if r1 == 0 {
		return nil, nil, fmt.Errorf("failed to determine size of buffer needed: %v", err)
	}
	if r1 < rawSMBIOSDataHeaderSize {
		return nil, nil, fmt.Errorf("reported buffer size smaller than expected: reported %d, expected >= 8", r1)
	}

	bufferSize := uint32(r1)
	buffer := make([]byte, bufferSize)

	r1, _, err = procGetSystemFirmwareTable.Call(
		uintptr(firmwareTableProviderSigRSMB), // FirmwareTableProviderSignature = 'RSMB'
		0,                                     // FirmwareTableID = 0
		uintptr(unsafe.Pointer(&buffer[0])),   // pFirmwareTableBuffer = &buffer
		uintptr(bufferSize),                   // BufferSize = bufferSize
	)
	bytesWritten := uint32(r1)

	// Check for the two possible failure cases documented in API:
	if bytesWritten > bufferSize {
		return nil, nil, fmt.Errorf("buffer size was too small, somehow: have %d bytes, Windows wanted %d bytes", bufferSize, bytesWritten)
	}
	if bytesWritten == 0 {
		return nil, nil, fmt.Errorf("failed to read SMBIOS data: %v", err)
	}

	// At this point, bytesWritten <= bufferSize, which means the call succeeded as
	// per the MSDN documentation.

	return windowsStream(buffer[:bytesWritten])
}
