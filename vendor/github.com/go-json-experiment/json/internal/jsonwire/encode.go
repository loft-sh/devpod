// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonwire

import (
	"math"
	"slices"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/go-json-experiment/json/internal/jsonflags"
)

// escapeASCII reports whether the ASCII character needs to be escaped.
// It conservatively assumes EscapeForHTML.
var escapeASCII = [...]uint8{
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // escape control characters
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // escape control characters
	0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, // escape '"' and '&'
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, // escape '<' and '>'
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, // escape '\\'
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
}

// NeedEscape reports whether src needs escaping of any characters.
// It conservatively assumes EscapeForHTML and EscapeForJS.
// It reports true for inputs with invalid UTF-8.
func NeedEscape[Bytes ~[]byte | ~string](src Bytes) bool {
	var i int
	for uint(len(src)) > uint(i) {
		if c := src[i]; c < utf8.RuneSelf {
			if escapeASCII[c] > 0 {
				return true
			}
			i++
		} else {
			r, rn := utf8.DecodeRuneInString(string(truncateMaxUTF8(src[i:])))
			if r == utf8.RuneError || r == '\u2028' || r == '\u2029' {
				return true
			}
			i += rn
		}
	}
	return false
}

// AppendQuote appends src to dst as a JSON string per RFC 7159, section 7.
//
// It takes in flags and respects the following:
//   - EscapeForHTML escapes '<', '>', and '&'.
//   - EscapeForJS escapes '\u2028' and '\u2029'.
//   - AllowInvalidUTF8 avoids reporting an error for invalid UTF-8.
//
// Regardless of whether AllowInvalidUTF8 is specified,
// invalid bytes are replaced with the Unicode replacement character ('\ufffd').
// If no escape flags are set, then the shortest representable form is used,
// which is also the canonical form for strings (RFC 8785, section 3.2.2.2).
func AppendQuote[Bytes ~[]byte | ~string](dst []byte, src Bytes, flags *jsonflags.Flags) ([]byte, error) {
	var i, n int
	var hasInvalidUTF8 bool
	dst = slices.Grow(dst, len(`"`)+len(src)+len(`"`))
	dst = append(dst, '"')
	for uint(len(src)) > uint(n) {
		// Handle single-byte ASCII.
		if c := src[n]; c < utf8.RuneSelf {
			n++
			if escapeASCII[c] > 0 {
				if (c == '<' || c == '>' || c == '&') && !flags.Get(jsonflags.EscapeForHTML) {
					continue
				}
				dst = append(dst, src[i:n-1]...)
				dst = appendEscapedASCII(dst, c)
				i = n
			}
			continue
		}

		// Handle multi-byte Unicode.
		switch r, rn := utf8.DecodeRuneInString(string(truncateMaxUTF8(src[n:]))); {
		case r == utf8.RuneError && rn == 1:
			hasInvalidUTF8 = true
			dst = append(dst, src[i:n]...)
			dst = append(dst, "\ufffd"...)
			n += rn
			i = n
		case (r == '\u2028' || r == '\u2029') && flags.Get(jsonflags.EscapeForJS):
			dst = append(dst, src[i:n]...)
			dst = appendEscapedUnicode(dst, r)
			n += rn
			i = n
		default:
			n += rn
		}
	}
	dst = append(dst, src[i:n]...)
	dst = append(dst, '"')
	if hasInvalidUTF8 && !flags.Get(jsonflags.AllowInvalidUTF8) {
		return dst, ErrInvalidUTF8
	}
	return dst, nil
}

func appendEscapedASCII(dst []byte, c byte) []byte {
	switch c {
	case '"', '\\':
		dst = append(dst, '\\', c)
	case '\b':
		dst = append(dst, "\\b"...)
	case '\f':
		dst = append(dst, "\\f"...)
	case '\n':
		dst = append(dst, "\\n"...)
	case '\r':
		dst = append(dst, "\\r"...)
	case '\t':
		dst = append(dst, "\\t"...)
	default:
		dst = appendEscapedUTF16(dst, uint16(c))
	}
	return dst
}

func appendEscapedUnicode(dst []byte, r rune) []byte {
	if r1, r2 := utf16.EncodeRune(r); r1 != '\ufffd' && r2 != '\ufffd' {
		dst = appendEscapedUTF16(dst, uint16(r1))
		dst = appendEscapedUTF16(dst, uint16(r2))
	} else {
		dst = appendEscapedUTF16(dst, uint16(r))
	}
	return dst
}

func appendEscapedUTF16(dst []byte, x uint16) []byte {
	const hex = "0123456789abcdef"
	return append(dst, '\\', 'u', hex[(x>>12)&0xf], hex[(x>>8)&0xf], hex[(x>>4)&0xf], hex[(x>>0)&0xf])
}

// ReformatString consumes a JSON string from src and appends it to dst,
// reformatting it if necessary for the given escapeRune parameter.
// It returns the appended output and the number of consumed input bytes.
func ReformatString(dst, src []byte, flags *jsonflags.Flags) ([]byte, int, error) {
	// TODO: Should this update ValueFlags as input?
	var valFlags ValueFlags
	n, err := ConsumeString(&valFlags, src, !flags.Get(jsonflags.AllowInvalidUTF8))
	if err != nil {
		return dst, n, err
	}
	isCanonical := !flags.Get(jsonflags.EscapeForHTML | jsonflags.EscapeForJS)
	if flags.Get(jsonflags.PreserveRawStrings) || (isCanonical && valFlags.IsCanonical()) {
		dst = append(dst, src[:n]...) // copy the string verbatim
		return dst, n, nil
	}

	// TODO: Implement a direct, raw-to-raw reformat for strings.
	// If the escapeRune option would have resulted in no changes to the output,
	// it would be faster to simply append src to dst without going through
	// an intermediary representation in a separate buffer.
	b, _ := AppendUnquote(nil, src[:n])
	dst, _ = AppendQuote(dst, string(b), flags)
	return dst, n, nil
}

// AppendFloat appends src to dst as a JSON number per RFC 7159, section 6.
// It formats numbers similar to the ES6 number-to-string conversion.
// See https://go.dev/issue/14135.
//
// The output is identical to ECMA-262, 6th edition, section 7.1.12.1 and with
// RFC 8785, section 3.2.2.3 for 64-bit floating-point numbers except for -0,
// which is formatted as -0 instead of just 0.
//
// For 32-bit floating-point numbers,
// the output is a 32-bit equivalent of the algorithm.
// Note that ECMA-262 specifies no algorithm for 32-bit numbers.
func AppendFloat(dst []byte, src float64, bits int) []byte {
	if bits == 32 {
		src = float64(float32(src))
	}

	abs := math.Abs(src)
	fmt := byte('f')
	if abs != 0 {
		if bits == 64 && (float64(abs) < 1e-6 || float64(abs) >= 1e21) ||
			bits == 32 && (float32(abs) < 1e-6 || float32(abs) >= 1e21) {
			fmt = 'e'
		}
	}
	dst = strconv.AppendFloat(dst, src, fmt, -1, bits)
	if fmt == 'e' {
		// Clean up e-09 to e-9.
		n := len(dst)
		if n >= 4 && dst[n-4] == 'e' && dst[n-3] == '-' && dst[n-2] == '0' {
			dst[n-2] = dst[n-1]
			dst = dst[:n-1]
		}
	}
	return dst
}

// ReformatNumber consumes a JSON string from src and appends it to dst,
// canonicalizing it if specified.
// It returns the appended output and the number of consumed input bytes.
func ReformatNumber(dst, src []byte, canonicalize bool) ([]byte, int, error) {
	n, err := ConsumeNumber(src)
	if err != nil {
		return dst, n, err
	}
	if !canonicalize {
		dst = append(dst, src[:n]...) // copy the number verbatim
		return dst, n, nil
	}

	// Canonicalize the number per RFC 8785, section 3.2.2.3.
	// As an optimization, we can copy integer numbers below 2⁵³ verbatim.
	const maxExactIntegerDigits = 16 // len(strconv.AppendUint(nil, 1<<53, 10))
	if n < maxExactIntegerDigits && ConsumeSimpleNumber(src[:n]) == n {
		dst = append(dst, src[:n]...) // copy the number verbatim
		return dst, n, nil
	}
	fv, _ := strconv.ParseFloat(string(src[:n]), 64)
	switch {
	case fv == 0:
		fv = 0 // normalize negative zero as just zero
	case math.IsInf(fv, +1):
		fv = +math.MaxFloat64
	case math.IsInf(fv, -1):
		fv = -math.MaxFloat64
	}
	return AppendFloat(dst, fv, 64), n, nil
}
