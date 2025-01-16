// Copyright 2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uio

import (
	"bytes"
	"errors"
	"io"

	"github.com/pierrec/lz4/v4"
)

const (
	// preReadSizeBytes is the num of bytes pre-read from a io.Reader that will
	// be used to match against archive header.
	defaultArchivePreReadSizeBytes = 1024
)

var ErrPreReadError = errors.New("pre-read nothing")

// ArchiveReader reads from a io.Reader, decompresses source bytes
// when applicable.
//
// It allows probing for multiple archive format, while still able
// to read from beginning, by pre-reading a small number of bytes.
//
// Always use newArchiveReader to initialize.
type ArchiveReader struct {
	// src is where we read source bytes.
	src io.Reader
	// buf stores pre-read bytes from original io.Reader. Archive format
	// detection will be done against it.
	buf []byte

	// preReadSizeBytes is how many bytes we pre-read for magic number
	// matching for each archive type. This should be greater than or
	// equal to the largest header frame size of each supported archive
	// format.
	preReadSizeBytes int
}

func NewArchiveReader(r io.Reader) (ArchiveReader, error) {
	ar := ArchiveReader{
		src: r,
		// Randomly chosen, should be enough for most types:
		//
		// e.g. gzip with 10 byte header, lz4 with a header size
		// between 7 and 19 bytes.
		preReadSizeBytes: defaultArchivePreReadSizeBytes,
	}
	pbuf := make([]byte, ar.preReadSizeBytes)

	nr, err := io.ReadFull(r, pbuf)
	// In case the image is smaller pre-read block size, 1kb for now.
	// Ever possible ? probably not in case a compression is needed!
	ar.buf = pbuf[:nr]
	if err == io.EOF {
		// If we could not pre-read anything, we can't determine if
		// it is a compressed file.
		ar.src = io.MultiReader(bytes.NewReader(pbuf[:nr]), r)
		return ar, ErrPreReadError
	}

	// Try each supported compression type, return upon first match.

	// Try lz4.
	// magic number error will be thrown if source is not a lz4 archive.
	// e.g. "lz4: bad magic number".
	if ok, err := lz4.ValidFrameHeader(ar.buf); err == nil && ok {
		ar.src = lz4.NewReader(io.MultiReader(bytes.NewReader(ar.buf), r))
		return ar, nil
	}

	// Try other archive types here, gzip, xz, etc when needed.

	// Last resort, read as is.
	ar.src = io.MultiReader(bytes.NewReader(ar.buf), r)
	return ar, nil
}

func (ar ArchiveReader) Read(p []byte) (n int, err error) {
	return ar.src.Read(p)
}
