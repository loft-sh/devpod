// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package jsonwire implements stateless functionality for handling JSON text.
package jsonwire

import (
	"cmp"
	"errors"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// TrimSuffixWhitespace trims JSON from the end of b.
func TrimSuffixWhitespace(b []byte) []byte {
	// NOTE: The arguments and logic are kept simple to keep this inlinable.
	n := len(b) - 1
	for n >= 0 && (b[n] == ' ' || b[n] == '\t' || b[n] == '\r' || b[n] == '\n') {
		n--
	}
	return b[:n+1]
}

// TrimSuffixString trims a valid JSON string at the end of b.
// The behavior is undefined if there is not a valid JSON string present.
func TrimSuffixString(b []byte) []byte {
	// NOTE: The arguments and logic are kept simple to keep this inlinable.
	if len(b) > 0 && b[len(b)-1] == '"' {
		b = b[:len(b)-1]
	}
	for len(b) >= 2 && !(b[len(b)-1] == '"' && b[len(b)-2] != '\\') {
		b = b[:len(b)-1] // trim all characters except an unescaped quote
	}
	if len(b) > 0 && b[len(b)-1] == '"' {
		b = b[:len(b)-1]
	}
	return b
}

// HasSuffixByte reports whether b ends with c.
func HasSuffixByte(b []byte, c byte) bool {
	// NOTE: The arguments and logic are kept simple to keep this inlinable.
	return len(b) > 0 && b[len(b)-1] == c
}

// TrimSuffixByte removes c from the end of b if it is present.
func TrimSuffixByte(b []byte, c byte) []byte {
	// NOTE: The arguments and logic are kept simple to keep this inlinable.
	if len(b) > 0 && b[len(b)-1] == c {
		return b[:len(b)-1]
	}
	return b
}

// QuoteRune quotes the first rune in the input.
func QuoteRune[Bytes ~[]byte | ~string](b Bytes) string {
	r, n := utf8.DecodeRuneInString(string(truncateMaxUTF8(b)))
	if r == utf8.RuneError && n == 1 {
		return `'\x` + strconv.FormatUint(uint64(b[0]), 16) + `'`
	}
	return strconv.QuoteRune(r)
}

// CompareUTF16 lexicographically compares x to y according
// to the UTF-16 codepoints of the UTF-8 encoded input strings.
// This implements the ordering specified in RFC 8785, section 3.2.3.
func CompareUTF16[Bytes ~[]byte | ~string](x, y Bytes) int {
	// NOTE: This is an optimized, mostly allocation-free implementation
	// of CompareUTF16Simple in wire_test.go. FuzzCompareUTF16 verifies that the
	// two implementations agree on the result of comparing any two strings.
	isUTF16Self := func(r rune) bool {
		return ('\u0000' <= r && r <= '\uD7FF') || ('\uE000' <= r && r <= '\uFFFF')
	}

	var invalidUTF8 bool
	x0, y0 := x, y
	for {
		if len(x) == 0 || len(y) == 0 {
			if len(x) == len(y) && invalidUTF8 {
				return strings.Compare(string(x0), string(y0))
			}
			return cmp.Compare(len(x), len(y))
		}

		// ASCII fast-path.
		if x[0] < utf8.RuneSelf || y[0] < utf8.RuneSelf {
			if x[0] != y[0] {
				return cmp.Compare(x[0], y[0])
			}
			x, y = x[1:], y[1:]
			continue
		}

		// Decode next pair of runes as UTF-8.
		rx, nx := utf8.DecodeRuneInString(string(truncateMaxUTF8(x)))
		ry, ny := utf8.DecodeRuneInString(string(truncateMaxUTF8(y)))

		selfx := isUTF16Self(rx)
		selfy := isUTF16Self(ry)
		switch {
		// The x rune is a single UTF-16 codepoint, while
		// the y rune is a surrogate pair of UTF-16 codepoints.
		case selfx && !selfy:
			ry, _ = utf16.EncodeRune(ry)
		// The y rune is a single UTF-16 codepoint, while
		// the x rune is a surrogate pair of UTF-16 codepoints.
		case selfy && !selfx:
			rx, _ = utf16.EncodeRune(rx)
		}
		if rx != ry {
			return cmp.Compare(rx, ry)
		}
		invalidUTF8 = invalidUTF8 || (rx == utf8.RuneError && nx == 1) || (ry == utf8.RuneError && ny == 1)
		x, y = x[nx:], y[ny:]
	}
}

// truncateMaxUTF8 truncates b such it contains at least one rune.
//
// The utf8 package currently lacks generic variants, which complicates
// generic functions that operates on either []byte or string.
// As a hack, we always call the utf8 function operating on strings,
// but always truncate the input such that the result is identical.
//
// Example usage:
//
//	utf8.DecodeRuneInString(string(truncateMaxUTF8(b)))
//
// Converting a []byte to a string is stack allocated since
// truncateMaxUTF8 guarantees that the []byte is short.
func truncateMaxUTF8[Bytes ~[]byte | ~string](b Bytes) Bytes {
	// TODO(https://go.dev/issue/56948): Remove this function and
	// instead directly call generic utf8 functions wherever used.
	if len(b) > utf8.UTFMax {
		return b[:utf8.UTFMax]
	}
	return b
}

// NewError and ErrInvalidUTF8 are injected by the "jsontext" package,
// so that these error types use the jsontext.SyntacticError type.
var (
	NewError       = errors.New
	ErrInvalidUTF8 = errors.New("invalid UTF-8 within string")
)

func NewInvalidCharacterError[Bytes ~[]byte | ~string](prefix Bytes, where string) error {
	what := QuoteRune(prefix)
	return NewError("invalid character " + what + " " + where)
}

func NewInvalidEscapeSequenceError[Bytes ~[]byte | ~string](what Bytes) error {
	label := "escape sequence"
	if len(what) > 6 {
		label = "surrogate pair"
	}
	needEscape := strings.IndexFunc(string(what), func(r rune) bool {
		return r == '`' || r == utf8.RuneError || unicode.IsSpace(r) || !unicode.IsPrint(r)
	}) >= 0
	if needEscape {
		return NewError("invalid " + label + " " + strconv.Quote(string(what)) + " within string")
	} else {
		return NewError("invalid " + label + " `" + string(what) + "` within string")
	}
}
