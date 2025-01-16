// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uio

import (
	"fmt"
	"io"
	"os"
)

// ReadOneByte reads one byte from given io.ReaderAt.
func ReadOneByte(r io.ReaderAt) error {
	buf := make([]byte, 1)
	n, err := r.ReadAt(buf, 0)
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("expected to read 1 byte, but got %d", n)
	}
	return nil
}

// LazyOpener is a lazy io.Reader.
//
// LazyOpener will use a given open function to derive an io.Reader when Read
// is first called on the LazyOpener.
type LazyOpener struct {
	r    io.Reader
	s    string
	err  error
	open func() (io.Reader, error)
}

// NewLazyOpener returns a lazy io.Reader based on `open`.
func NewLazyOpener(filename string, open func() (io.Reader, error)) *LazyOpener {
	if len(filename) == 0 {
		return nil
	}
	return &LazyOpener{s: filename, open: open}
}

// Read implements io.Reader.Read lazily.
//
// If called for the first time, the underlying reader will be obtained and
// then used for the first and subsequent calls to Read.
func (lr *LazyOpener) Read(p []byte) (int, error) {
	if lr.r == nil && lr.err == nil {
		lr.r, lr.err = lr.open()
	}
	if lr.err != nil {
		return 0, lr.err
	}
	return lr.r.Read(p)
}

// String implements fmt.Stringer.
func (lr *LazyOpener) String() string {
	if len(lr.s) > 0 {
		return lr.s
	}
	if lr.r != nil {
		return fmt.Sprintf("%v", lr.r)
	}
	return "unopened mystery file"
}

// Close implements io.Closer.Close.
func (lr *LazyOpener) Close() error {
	if c, ok := lr.r.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// LazyOpenerAt is a lazy io.ReaderAt.
//
// LazyOpenerAt will use a given open function to derive an io.ReaderAt when
// ReadAt is first called.
type LazyOpenerAt struct {
	r     io.ReaderAt
	s     string
	err   error
	limit int64
	open  func() (io.ReaderAt, error)
}

// NewLazyFile returns a lazy ReaderAt opened from path.
func NewLazyFile(path string) *LazyOpenerAt {
	if len(path) == 0 {
		return nil
	}
	return NewLazyOpenerAt(path, func() (io.ReaderAt, error) {
		return os.Open(path)
	})
}

// NewLazyLimitFile returns a lazy ReaderAt opened from path with a limit reader on it.
func NewLazyLimitFile(path string, limit int64) *LazyOpenerAt {
	if len(path) == 0 {
		return nil
	}
	return NewLazyLimitOpenerAt(path, limit, func() (io.ReaderAt, error) {
		return os.Open(path)
	})
}

// NewLazyOpenerAt returns a lazy io.ReaderAt based on `open`.
func NewLazyOpenerAt(filename string, open func() (io.ReaderAt, error)) *LazyOpenerAt {
	return &LazyOpenerAt{s: filename, open: open, limit: -1}
}

// NewLazyLimitOpenerAt returns a lazy io.ReaderAt based on `open`.
func NewLazyLimitOpenerAt(filename string, limit int64, open func() (io.ReaderAt, error)) *LazyOpenerAt {
	return &LazyOpenerAt{s: filename, open: open, limit: limit}
}

// String implements fmt.Stringer.
func (loa *LazyOpenerAt) String() string {
	if len(loa.s) > 0 {
		return loa.s
	}
	if loa.r != nil {
		return fmt.Sprintf("%v", loa.r)
	}
	return "unopened mystery file"
}

// File returns the backend file of the io.ReaderAt if it
// is backed by a os.File.
func (loa *LazyOpenerAt) File() *os.File {
	if f, ok := loa.r.(*os.File); ok {
		return f
	}
	return nil
}

// ReadAt implements io.ReaderAt.ReadAt.
func (loa *LazyOpenerAt) ReadAt(p []byte, off int64) (int, error) {
	if loa.r == nil && loa.err == nil {
		loa.r, loa.err = loa.open()
	}
	if loa.err != nil {
		return 0, loa.err
	}
	if loa.limit > 0 {
		if off >= loa.limit {
			return 0, io.EOF
		}
		if int64(len(p)) > loa.limit-off {
			p = p[0 : loa.limit-off]
		}
	}
	return loa.r.ReadAt(p, off)
}

// Close implements io.Closer.Close.
func (loa *LazyOpenerAt) Close() error {
	if c, ok := loa.r.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
