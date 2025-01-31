// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsontext

import (
	"strings"

	"github.com/go-json-experiment/json/internal/jsonflags"
	"github.com/go-json-experiment/json/internal/jsonopts"
	"github.com/go-json-experiment/json/internal/jsonwire"
)

// Options configures [NewEncoder], [Encoder.Reset], [NewDecoder],
// and [Decoder.Reset] with specific features.
// Each function takes in a variadic list of options, where properties
// set in latter options override the value of previously set properties.
//
// The Options type is identical to [encoding/json.Options] and
// [encoding/json/v2.Options]. Options from the other packages may
// be passed to functionality in this package, but are ignored.
// Options from this package may be used with the other packages.
type Options = jsonopts.Options

// AllowDuplicateNames specifies that JSON objects may contain
// duplicate member names. Disabling the duplicate name check may provide
// performance benefits, but breaks compliance with RFC 7493, section 2.3.
// The input or output will still be compliant with RFC 8259,
// which leaves the handling of duplicate names as unspecified behavior.
//
// This affects either encoding or decoding.
func AllowDuplicateNames(v bool) Options {
	if v {
		return jsonflags.AllowDuplicateNames | 1
	} else {
		return jsonflags.AllowDuplicateNames | 0
	}
}

// AllowInvalidUTF8 specifies that JSON strings may contain invalid UTF-8,
// which will be mangled as the Unicode replacement character, U+FFFD.
// This causes the encoder or decoder to break compliance with
// RFC 7493, section 2.1, and RFC 8259, section 8.1.
//
// This affects either encoding or decoding.
func AllowInvalidUTF8(v bool) Options {
	if v {
		return jsonflags.AllowInvalidUTF8 | 1
	} else {
		return jsonflags.AllowInvalidUTF8 | 0
	}
}

// EscapeForHTML specifies that '<', '>', and '&' characters within JSON strings
// should be escaped as a hexadecimal Unicode codepoint (e.g., \u003c) so that
// the output is safe to embed within HTML.
//
// This only affects encoding and is ignored when decoding.
func EscapeForHTML(v bool) Options {
	if v {
		return jsonflags.EscapeForHTML | 1
	} else {
		return jsonflags.EscapeForHTML | 0
	}
}

// EscapeForJS specifies that U+2028 and U+2029 characters within JSON strings
// should be escaped as a hexadecimal Unicode codepoint (e.g., \u2028) so that
// the output is valid to embed within JavaScript. See RFC 8259, section 12.
//
// This only affects encoding and is ignored when decoding.
func EscapeForJS(v bool) Options {
	if v {
		return jsonflags.EscapeForJS | 1
	} else {
		return jsonflags.EscapeForJS | 0
	}
}

// SpaceAfterColon specifies that the JSON output should emit a space character
// after each colon separator following a JSON object name.
// If false, then no space character appears after the colon separator.
//
// This only affects encoding and is ignored when decoding.
func SpaceAfterColon(v bool) Options {
	if v {
		return jsonflags.SpaceAfterColon | 1
	} else {
		return jsonflags.SpaceAfterColon | 0
	}
}

// SpaceAfterComma specifies that the JSON output should emit a space character
// after each comma separator following a JSON object value or array element.
// If false, then no space character appears after the comma separator.
//
// This only affects encoding and is ignored when decoding.
func SpaceAfterComma(v bool) Options {
	if v {
		return jsonflags.SpaceAfterComma | 1
	} else {
		return jsonflags.SpaceAfterComma | 0
	}
}

// Multiline specifies that the JSON output should expand to multiple lines,
// where every JSON object member or JSON array element appears on
// a new, indented line according to the nesting depth.
//
// If [SpaceAfterColon] is not specified, then the default is true.
// If [SpaceAfterComma] is not specified, then the default is false.
// If [WithIndent] is not specified, then the default is "\t".
//
// If set to false, then the output is a single-line,
// where the only whitespace emitted is determined by the current
// values of [SpaceAfterColon] and [SpaceAfterComma].
//
// This only affects encoding and is ignored when decoding.
func Multiline(v bool) Options {
	if v {
		return jsonflags.Multiline | 1
	} else {
		return jsonflags.Multiline | 0
	}
}

// WithIndent specifies that the encoder should emit multiline output
// where each element in a JSON object or array begins on a new, indented line
// beginning with the indent prefix (see [WithIndentPrefix])
// followed by one or more copies of indent according to the nesting depth.
// The indent must only be composed of space or tab characters.
//
// If the intent to emit indented output without a preference for
// the particular indent string, then use [Multiline] instead.
//
// This only affects encoding and is ignored when decoding.
// Use of this option implies [Multiline] being set to true.
func WithIndent(indent string) Options {
	// Fast-path: Return a constant for common indents, which avoids allocating.
	// These are derived from analyzing the Go module proxy on 2023-07-01.
	switch indent {
	case "\t":
		return jsonopts.Indent("\t") // ~14k usages
	case "    ":
		return jsonopts.Indent("    ") // ~18k usages
	case "   ":
		return jsonopts.Indent("   ") // ~1.7k usages
	case "  ":
		return jsonopts.Indent("  ") // ~52k usages
	case " ":
		return jsonopts.Indent(" ") // ~12k usages
	case "":
		return jsonopts.Indent("") // ~1.5k usages
	}

	// Otherwise, allocate for this unique value.
	if s := strings.Trim(indent, " \t"); len(s) > 0 {
		panic("json: invalid character " + jsonwire.QuoteRune(s) + " in indent")
	}
	return jsonopts.Indent(indent)
}

// WithIndentPrefix specifies that the encoder should emit multiline output
// where each element in a JSON object or array begins on a new, indented line
// beginning with the indent prefix followed by one or more copies of indent
// (see [WithIndent]) according to the nesting depth.
// The prefix must only be composed of space or tab characters.
//
// This only affects encoding and is ignored when decoding.
// Use of this option implies [Multiline] being set to true.
func WithIndentPrefix(prefix string) Options {
	if s := strings.Trim(prefix, " \t"); len(s) > 0 {
		panic("json: invalid character " + jsonwire.QuoteRune(s) + " in indent prefix")
	}
	return jsonopts.IndentPrefix(prefix)
}

/*
// TODO(https://go.dev/issue/56733): Implement WithByteLimit and WithDepthLimit.

// WithByteLimit sets a limit on the number of bytes of input or output bytes
// that may be consumed or produced for each top-level JSON value.
// If a [Decoder] or [Encoder] method call would need to consume/produce
// more than a total of n bytes to make progress on the top-level JSON value,
// then the call will report an error.
// Whitespace before and within the top-level value are counted against the limit.
// Whitespace after a top-level value are counted against the limit
// for the next top-level value.
//
// A non-positive limit is equivalent to no limit at all.
// If unspecified, the default limit is no limit at all.
func WithByteLimit(n int64) Options {
	return jsonopts.ByteLimit(max(n, 0))
}

// WithDepthLimit sets a limit on the maximum depth of JSON nesting
// that may be consumed or produced for each top-level JSON value.
// If a [Decoder] or [Encoder] method call would need to consume or produce
// a depth greater than n to make progress on the top-level JSON value,
// then the call will report an error.
//
// A non-positive limit is equivalent to no limit at all.
// If unspecified, the default limit is 10000.
func WithDepthLimit(n int) Options {
	return jsonopts.DepthLimit(max(n, 0))
}
*/
