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
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
)

const (
	// headerLen is the length of the Header structure.
	headerLen = 4

	// typeEndOfTable indicates the end of a stream of Structures.
	typeEndOfTable = 127
)

var (
	// Byte slices used to help parsing string-sets.
	null         = []byte{0x00}
	endStringSet = []byte{0x00, 0x00}
)

// A Decoder decodes Structures from a stream.
type Decoder struct {
	br *bufio.Reader
	b  []byte
}

// Stream locates and opens a stream of SMBIOS data and the SMBIOS entry
// point from an operating system-specific location.  The stream must be
// closed after decoding to free its resources.
//
// If no suitable location is found, an error is returned.
func Stream() (io.ReadCloser, EntryPoint, error) {
	rc, ep, err := stream()
	if err != nil {
		return nil, nil, err
	}

	// The io.ReadCloser from stream could be any one of a number of types
	// depending on the source of the SMBIOS stream information.
	//
	// To prevent the caller from potentially tampering with something dangerous
	// like mmap'd memory by using a type assertion, we make the io.ReadCloser
	// into an opaque and unexported type to prevent type assertion.
	return &opaqueReadCloser{rc: rc}, ep, nil
}

// NewDecoder creates a Decoder which decodes Structures from the input stream.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		br: bufio.NewReader(r),
		b:  make([]byte, 1024),
	}
}

// Decode decodes Structures from the Decoder's stream until an End-of-table
// structure is found.
func (d *Decoder) Decode() ([]*Structure, error) {
	var ss []*Structure

	for {
		s, err := d.next()
		if err != nil {
			return nil, err
		}

		// End-of-table structure indicates end of stream.
		ss = append(ss, s)
		if s.Header.Type == typeEndOfTable {
			break
		}
	}

	return ss, nil
}

// next decodes the next Structure from the stream.
func (d *Decoder) next() (*Structure, error) {
	h, err := d.parseHeader()
	if err != nil {
		return nil, err
	}

	// Length of formatted section is length specified by header, minus
	// the length of the header itself.
	l := int(h.Length) - headerLen
	fb, err := d.parseFormatted(l)
	if err != nil {
		return nil, err
	}

	ss, err := d.parseStrings()
	if err != nil {
		return nil, err
	}

	return &Structure{
		Header:    *h,
		Formatted: fb,
		Strings:   ss,
	}, nil
}

// parseHeader parses a Structure's Header from the stream.
func (d *Decoder) parseHeader() (*Header, error) {
	if _, err := io.ReadFull(d.br, d.b[:headerLen]); err != nil {
		return nil, err
	}

	return &Header{
		Type:   d.b[0],
		Length: d.b[1],
		Handle: binary.LittleEndian.Uint16(d.b[2:4]),
	}, nil
}

// parseFormatted parses a Structure's formatted data from the stream.
func (d *Decoder) parseFormatted(l int) ([]byte, error) {
	// Guard against malformed input length.
	if l < 0 {
		return nil, io.ErrUnexpectedEOF
	}
	if l == 0 {
		// No formatted data.
		return nil, nil
	}

	if _, err := io.ReadFull(d.br, d.b[:l]); err != nil {
		return nil, err
	}

	// Make a copy to free up the internal buffer.
	fb := make([]byte, len(d.b[:l]))
	copy(fb, d.b[:l])

	return fb, nil
}

// parseStrings parses a Structure's strings from the stream, if they
// are present.
func (d *Decoder) parseStrings() ([]string, error) {
	term, err := d.br.Peek(2)
	if err != nil {
		return nil, err
	}

	// If no string-set present, discard delimeter and end parsing.
	if bytes.Equal(term, endStringSet) {
		if _, err := d.br.Discard(2); err != nil {
			return nil, err
		}

		return nil, nil
	}

	var ss []string
	for {
		s, more, err := d.parseString()
		if err != nil {
			return nil, err
		}

		// When final string is received, end parse loop.
		ss = append(ss, s)
		if !more {
			break
		}
	}

	return ss, nil
}

// parseString parses a single string from the stream, and returns if
// any more strings are present.
func (d *Decoder) parseString() (str string, more bool, err error) {
	// We initially read bytes because it's more efficient to manipulate bytes
	// and allocate a string once we're all done.
	//
	// Strings are null-terminated.
	raw, err := d.br.ReadBytes(0x00)
	if err != nil {
		return "", false, err
	}

	b := bytes.TrimRight(raw, "\x00")

	peek, err := d.br.Peek(1)
	if err != nil {
		return "", false, err
	}

	if !bytes.Equal(peek, null) {
		// Next byte isn't null; more strings to come.
		return string(b), true, nil
	}

	// If two null bytes appear in a row, end of string-set.
	// Discard the null and indicate no more strings.
	if _, err := d.br.Discard(1); err != nil {
		return "", false, err
	}

	return string(b), false, nil
}

var _ io.ReadCloser = &opaqueReadCloser{}

// An opaqueReadCloser masks the type of the underlying io.ReadCloser to
// prevent type assertions.
type opaqueReadCloser struct {
	rc io.ReadCloser
}

func (rc *opaqueReadCloser) Read(b []byte) (int, error) { return rc.rc.Read(b) }
func (rc *opaqueReadCloser) Close() error               { return rc.rc.Close() }
