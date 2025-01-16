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
	"fmt"
	"io"
	"io/ioutil"
)

// Anchor strings used to detect entry points.
var (
	// Used when searching for an entry point in memory.
	magicPrefix = []byte("_SM")

	// Used to determine specific entry point types.
	magic32  = []byte("_SM_")
	magic64  = []byte("_SM3_")
	magicDMI = []byte("_DMI_")
)

// An EntryPoint is an SMBIOS entry point.  EntryPoints contain various
// properties about SMBIOS.
//
// Use a type assertion to access detailed EntryPoint information.
type EntryPoint interface {
	// Table returns the memory address and maximum size of the SMBIOS table.
	Table() (address, size int)

	// Version returns the system's SMBIOS version.
	Version() (major, minor, revision int)
}

// ParseEntryPoint parses an EntryPoint from the input stream.
func ParseEntryPoint(r io.Reader) (EntryPoint, error) {
	// Prevent unbounded reads since this structure should be small.
	b, err := ioutil.ReadAll(io.LimitReader(r, 64))
	if err != nil {
		return nil, err
	}

	if l := len(b); l < 4 {
		return nil, fmt.Errorf("too few bytes for SMBIOS entry point magic: %d", l)
	}

	switch {
	case bytes.HasPrefix(b, magic32):
		return parse32(b)
	case bytes.HasPrefix(b, magic64):
		return parse64(b)
	}

	return nil, fmt.Errorf("unrecognized SMBIOS entry point magic: %v", b[0:4])
}

var _ EntryPoint = &EntryPoint32Bit{}

// EntryPoint32Bit is the SMBIOS 32-bit Entry Point structure, used starting
// in SMBIOS 2.1.
type EntryPoint32Bit struct {
	Anchor                string
	Checksum              uint8
	Length                uint8
	Major                 uint8
	Minor                 uint8
	MaxStructureSize      uint16
	EntryPointRevision    uint8
	FormattedArea         [5]byte
	IntermediateAnchor    string
	IntermediateChecksum  uint8
	StructureTableLength  uint16
	StructureTableAddress uint32
	NumberStructures      uint16
	BCDRevision           uint8
}

// Table implements EntryPoint.
func (e *EntryPoint32Bit) Table() (address, size int) {
	return int(e.StructureTableAddress), int(e.StructureTableLength)
}

// Version implements EntryPoint.
func (e *EntryPoint32Bit) Version() (major, minor, revision int) {
	return int(e.Major), int(e.Minor), 0
}

// parse32 parses an EntryPoint32Bit from b.
func parse32(b []byte) (*EntryPoint32Bit, error) {
	l := len(b)

	// Correct minimum length as of SMBIOS 3.1.1.
	const expLen = 31
	if l < expLen {
		return nil, fmt.Errorf("expected SMBIOS 32-bit entry point minimum length of at least %d, but got: %d", expLen, l)
	}

	// Allow more data in the buffer than the actual length, for when the
	// entry point is being read from system memory.
	length := b[5]
	if l < int(length) {
		return nil, fmt.Errorf("expected SMBIOS 32-bit entry point actual length of at least %d, but got: %d", length, l)
	}

	// Look for intermediate anchor with DMI magic.
	iAnchor := b[16:21]
	if !bytes.Equal(iAnchor, magicDMI) {
		return nil, fmt.Errorf("incorrect DMI magic in SMBIOS 32-bit entry point: %v", iAnchor)
	}

	// Entry point checksum occurs at index 4, compute and verify it.
	const epChkIndex = 4
	epChk := b[epChkIndex]
	if err := checksum(epChk, epChkIndex, b[:length]); err != nil {
		return nil, err
	}

	// Since we already computed the checksum for the outer entry point,
	// no real need to compute it for the intermediate entry point.

	ep := &EntryPoint32Bit{
		Anchor:                string(b[0:4]),
		Checksum:              epChk,
		Length:                length,
		Major:                 b[6],
		Minor:                 b[7],
		MaxStructureSize:      binary.LittleEndian.Uint16(b[8:10]),
		EntryPointRevision:    b[10],
		IntermediateAnchor:    string(iAnchor),
		IntermediateChecksum:  b[21],
		StructureTableLength:  binary.LittleEndian.Uint16(b[22:24]),
		StructureTableAddress: binary.LittleEndian.Uint32(b[24:28]),
		NumberStructures:      binary.LittleEndian.Uint16(b[28:30]),
		BCDRevision:           b[30],
	}
	copy(ep.FormattedArea[:], b[10:15])

	return ep, nil
}

var _ EntryPoint = &EntryPoint64Bit{}

// EntryPoint64Bit is the SMBIOS 64-bit Entry Point structure, used starting
// in SMBIOS 3.0.
type EntryPoint64Bit struct {
	Anchor                string
	Checksum              uint8
	Length                uint8
	Major                 uint8
	Minor                 uint8
	Revision              uint8
	EntryPointRevision    uint8
	Reserved              uint8
	StructureTableMaxSize uint32
	StructureTableAddress uint64
}

// Table implements EntryPoint.
func (e *EntryPoint64Bit) Table() (address, size int) {
	return int(e.StructureTableAddress), int(e.StructureTableMaxSize)
}

// Version implements EntryPoint.
func (e *EntryPoint64Bit) Version() (major, minor, revision int) {
	return int(e.Major), int(e.Minor), int(e.Revision)
}

const (
	// expLen64 is the expected minimum length of a 64-bit entry point.
	// Correct minimum length as of SMBIOS 3.1.1.
	expLen64 = 24

	// chkIndex64 is the index of the checksum byte in a 64-bit entry point.
	chkIndex64 = 5
)

// parse64 parses an EntryPoint64Bit from b.
func parse64(b []byte) (*EntryPoint64Bit, error) {
	l := len(b)

	// Ensure expected minimum length.
	if l < expLen64 {
		return nil, fmt.Errorf("expected SMBIOS 64-bit entry point minimum length of at least %d, but got: %d", expLen64, l)
	}

	// Allow more data in the buffer than the actual length, for when the
	// entry point is being read from system memory.
	length := b[6]
	if l < int(length) {
		return nil, fmt.Errorf("expected SMBIOS 64-bit entry point actual length of at least %d, but got: %d", length, l)
	}

	// Checksum occurs at index 5, compute and verify it.
	chk := b[chkIndex64]
	if err := checksum(chk, chkIndex64, b); err != nil {
		return nil, err
	}

	return &EntryPoint64Bit{
		Anchor:                string(b[0:5]),
		Checksum:              chk,
		Length:                length,
		Major:                 b[7],
		Minor:                 b[8],
		Revision:              b[9],
		EntryPointRevision:    b[10],
		Reserved:              b[11],
		StructureTableMaxSize: binary.LittleEndian.Uint32(b[12:16]),
		StructureTableAddress: binary.LittleEndian.Uint64(b[16:24]),
	}, nil
}

// checksum computes the checksum of b using the starting value of start, and
// skipping the checksum byte which occurs at index chkIndex.
//
// checksum assumes that b has already had its bounds checked.
func checksum(start uint8, chkIndex int, b []byte) error {
	chk := start
	for i := range b {
		// Checksum computation does not include index of checksum byte.
		if i == chkIndex {
			continue
		}

		chk += b[i]
	}

	if chk != 0 {
		return fmt.Errorf("invalid entry point checksum %#02x from initial checksum %#02x", chk, start)
	}

	return nil
}

// WindowsEntryPoint contains SMBIOS Table entry point data returned from
// GetSystemFirmwareTable. As raw access to the underlying memory is not given,
// the full breadth of information is not available.
type WindowsEntryPoint struct {
	Size         uint32
	MajorVersion byte
	MinorVersion byte
	Revision     byte
}

// Table implements EntryPoint. The returned address will always be 0, as it
// is not returned by GetSystemFirmwareTable.
func (e *WindowsEntryPoint) Table() (address, size int) {
	return 0, int(e.Size)
}

// Version implements EntryPoint.
func (e *WindowsEntryPoint) Version() (major, minor, revision int) {
	return int(e.MajorVersion), int(e.MinorVersion), int(e.Revision)
}
