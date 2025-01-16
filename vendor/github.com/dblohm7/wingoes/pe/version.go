// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

//go:build windows

package pe

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	errFixedFileInfoBadSig   = errors.New("bad VS_FIXEDFILEINFO signature")
	errFixedFileInfoTooShort = errors.New("buffer smaller than VS_FIXEDFILEINFO")
)

// VersionNumber encapsulates a four-component version number that is stored
// in Windows VERSIONINFO resources.
type VersionNumber struct {
	Major uint16
	Minor uint16
	Patch uint16
	Build uint16
}

func (vn VersionNumber) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", vn.Major, vn.Minor, vn.Patch, vn.Build)
}

type langAndCodePage struct {
	language uint16
	codePage uint16
}

// VersionInfo encapsulates a buffer containing the VERSIONINFO resources that
// have been successfully extracted from a PE binary.
type VersionInfo struct {
	buf            []byte
	fixed          *windows.VS_FIXEDFILEINFO
	translationIDs []langAndCodePage
}

const (
	langEnUS        = 0x0409
	codePageUTF16LE = 0x04B0
	langNeutral     = 0
	codePageNeutral = 0
)

// NewVersionInfo extracts any VERSIONINFO resource from filepath, parses its
// fixed-size information, and returns a *VersionInfo for further querying.
// It returns ErrNotPresent if no VERSIONINFO resources are found.
func NewVersionInfo(filepath string) (*VersionInfo, error) {
	bufSize, err := windows.GetFileVersionInfoSize(filepath, nil)
	if err != nil {
		if errors.Is(err, windows.ERROR_RESOURCE_TYPE_NOT_FOUND) {
			err = ErrNotPresent
		}
		return nil, err
	}

	buf := make([]byte, bufSize)
	if err := windows.GetFileVersionInfo(filepath, 0, bufSize, unsafe.Pointer(unsafe.SliceData(buf))); err != nil {
		return nil, err
	}

	var fixed *windows.VS_FIXEDFILEINFO
	var fixedLen uint32
	if err := windows.VerQueryValue(unsafe.Pointer(unsafe.SliceData(buf)), `\`, unsafe.Pointer(&fixed), &fixedLen); err != nil {
		return nil, err
	}
	if fixedLen < uint32(unsafe.Sizeof(windows.VS_FIXEDFILEINFO{})) {
		return nil, errFixedFileInfoTooShort
	}
	if fixed.Signature != 0xFEEF04BD {
		return nil, errFixedFileInfoBadSig
	}

	return &VersionInfo{
		buf:   buf,
		fixed: fixed,
	}, nil
}

func (vi *VersionInfo) VersionNumber() VersionNumber {
	f := vi.fixed

	return VersionNumber{
		Major: uint16(f.FileVersionMS >> 16),
		Minor: uint16(f.FileVersionMS & 0xFFFF),
		Patch: uint16(f.FileVersionLS >> 16),
		Build: uint16(f.FileVersionLS & 0xFFFF),
	}
}

func (vi *VersionInfo) maybeLoadTranslationIDs() {
	if vi.translationIDs != nil {
		// Already loaded
		return
	}

	// Preferred translations, in order of preference.
	preferredTranslationIDs := []langAndCodePage{
		langAndCodePage{
			language: langEnUS,
			codePage: codePageUTF16LE,
		},
		langAndCodePage{
			language: langNeutral,
			codePage: codePageNeutral,
		},
	}

	var ids *langAndCodePage
	var idsNumBytes uint32
	if err := windows.VerQueryValue(
		unsafe.Pointer(unsafe.SliceData(vi.buf)),
		`\VarFileInfo\Translation`,
		unsafe.Pointer(&ids),
		&idsNumBytes,
	); err != nil {
		// If nothing is listed, then just try to use our preferred translation IDs.
		vi.translationIDs = preferredTranslationIDs
		return
	}

	idsSlice := unsafe.Slice(ids, idsNumBytes/uint32(unsafe.Sizeof(*ids)))
	vi.translationIDs = append(preferredTranslationIDs, idsSlice...)
}

func (vi *VersionInfo) queryWithLangAndCodePage(key string, lcp langAndCodePage) (string, error) {
	fq := fmt.Sprintf("\\StringFileInfo\\%04x%04x\\%s", lcp.language, lcp.codePage, key)

	var value *uint16
	var valueLen uint32
	if err := windows.VerQueryValue(unsafe.Pointer(unsafe.SliceData(vi.buf)), fq, unsafe.Pointer(&value), &valueLen); err != nil {
		return "", err
	}

	return windows.UTF16ToString(unsafe.Slice(value, valueLen)), nil
}

// Field queries the version information for a field named key and either
// returns the field's value, or an error. It attempts to resolve strings using
// the following order of language preference: en-US, language-neutral, followed
// by the first entry in version info's list of supported languages that
// successfully resolves the key.
// If the key cannot be resolved, it returns ErrNotPresent.
func (vi *VersionInfo) Field(key string) (string, error) {
	vi.maybeLoadTranslationIDs()

	for _, lcp := range vi.translationIDs {
		value, err := vi.queryWithLangAndCodePage(key, lcp)
		if err == nil {
			return value, nil
		}
		if !errors.Is(err, windows.ERROR_RESOURCE_TYPE_NOT_FOUND) {
			return "", err
		}
		// Otherwise we continue looping and try the next language
	}

	return "", ErrNotPresent
}
