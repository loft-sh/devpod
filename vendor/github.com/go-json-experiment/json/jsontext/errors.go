// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsontext

import (
	"github.com/go-json-experiment/json/internal/jsonwire"
)

const errorPrefix = "jsontext: "

type ioError struct {
	action string // either "read" or "write"
	err    error
}

func (e *ioError) Error() string {
	return errorPrefix + e.action + " error: " + e.err.Error()
}
func (e *ioError) Unwrap() error {
	return e.err
}

// SyntacticError is a description of a syntactic error that occurred when
// encoding or decoding JSON according to the grammar.
//
// The contents of this error as produced by this package may change over time.
type SyntacticError struct {
	requireKeyedLiterals
	nonComparable

	// ByteOffset indicates that an error occurred after this byte offset.
	ByteOffset int64
	str        string
}

func (e *SyntacticError) Error() string {
	return errorPrefix + e.str
}
func (e *SyntacticError) withOffset(pos int64) error {
	return &SyntacticError{ByteOffset: pos, str: e.str}
}

func newDuplicateNameError[Bytes ~[]byte | ~string](quoted Bytes) *SyntacticError {
	return &SyntacticError{str: "duplicate name " + string(quoted) + " in object"}
}

func newInvalidCharacterError[Bytes ~[]byte | ~string](prefix Bytes, where string) *SyntacticError {
	what := jsonwire.QuoteRune(prefix)
	return &SyntacticError{str: "invalid character " + what + " " + where}
}

// TODO: Error types between "json", "jsontext", and "jsonwire" is a mess.
// Clean this up.
func init() {
	// Inject behavior in "jsonwire" so that it can produce SyntacticError types.
	jsonwire.NewError = func(s string) error { return &SyntacticError{str: s} }
	jsonwire.ErrInvalidUTF8 = &SyntacticError{str: jsonwire.ErrInvalidUTF8.Error()}
}
