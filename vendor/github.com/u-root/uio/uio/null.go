// Copyright 2012-2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Discard implementation copied from the Go project:
// https://golang.org/src/io/ioutil/ioutil.go.
// Copyright 2009 The Go Authors. All rights reserved.

package uio

import (
	"io"
	"sync"
)

// devNull implements an io.Writer and io.ReaderFrom that discards any writes.
type devNull struct{}

// devNull implements ReaderFrom as an optimization so io.Copy to
// ioutil.Discard can avoid doing unnecessary work.
var _ io.ReaderFrom = devNull{}

// Write is an io.Writer.Write that discards data.
func (devNull) Write(p []byte) (int, error) {
	return len(p), nil
}

// Name is like os.File.Name() and returns "null".
func (devNull) Name() string {
	return "null"
}

// WriteString implements io.StringWriter and discards given data.
func (devNull) WriteString(s string) (int, error) {
	return len(s), nil
}

var blackHolePool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 8192)
		return &b
	},
}

// ReadFrom implements io.ReaderFrom and discards data being read.
func (devNull) ReadFrom(r io.Reader) (n int64, err error) {
	bufp := blackHolePool.Get().(*[]byte)
	var readSize int
	for {
		readSize, err = r.Read(*bufp)
		n += int64(readSize)
		if err != nil {
			blackHolePool.Put(bufp)
			if err == io.EOF {
				return n, nil
			}
			return
		}
	}
}

// Close does nothing.
func (devNull) Close() error {
	return nil
}

// WriteNameCloser is the interface that groups Write, Close, and Name methods.
type WriteNameCloser interface {
	io.Writer
	io.Closer
	Name() string
}

// Discard is a WriteNameCloser on which all Write and Close calls succeed
// without doing anything, and the Name call returns "null".
var Discard WriteNameCloser = devNull{}
