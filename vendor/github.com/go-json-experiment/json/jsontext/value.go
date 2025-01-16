// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsontext

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"strings"
	"sync"

	"github.com/go-json-experiment/json/internal/jsonflags"
	"github.com/go-json-experiment/json/internal/jsonwire"
)

// NOTE: Value is analogous to v1 json.RawMessage.

// Value represents a single raw JSON value, which may be one of the following:
//   - a JSON literal (i.e., null, true, or false)
//   - a JSON string (e.g., "hello, world!")
//   - a JSON number (e.g., 123.456)
//   - an entire JSON object (e.g., {"fizz":"buzz"} )
//   - an entire JSON array (e.g., [1,2,3] )
//
// Value can represent entire array or object values, while [Token] cannot.
// Value may contain leading and/or trailing whitespace.
type Value []byte

// Clone returns a copy of v.
func (v Value) Clone() Value {
	return bytes.Clone(v)
}

// String returns the string formatting of v.
func (v Value) String() string {
	if v == nil {
		return "null"
	}
	return string(v)
}

// IsValid reports whether the raw JSON value is syntactically valid
// according to RFC 7493.
//
// It verifies whether the input is properly encoded as UTF-8,
// that escape sequences within strings decode to valid Unicode codepoints, and
// that all names in each object are unique.
// It does not verify whether numbers are representable within the limits
// of any common numeric type (e.g., float64, int64, or uint64).
func (v Value) IsValid() bool {
	d := getBufferedDecoder(v)
	defer putBufferedDecoder(d)
	_, errVal := d.ReadValue()
	_, errEOF := d.ReadToken()
	return errVal == nil && errEOF == io.EOF
}

// Compact removes all whitespace from the raw JSON value.
//
// It does not reformat JSON strings to use any other representation.
// It is guaranteed to succeed if the input is valid.
// If the value is already compacted, then the buffer is not mutated.
func (v *Value) Compact() error {
	return v.reformat(false, false, "", "")
}

// Indent reformats the whitespace in the raw JSON value so that each element
// in a JSON object or array begins on a new, indented line beginning with
// prefix followed by one or more copies of indent according to the nesting.
// The value does not begin with the prefix nor any indention,
// to make it easier to embed inside other formatted JSON data.
//
// It does not reformat JSON strings to use any other representation.
// It is guaranteed to succeed if the input is valid.
// If the value is already indented properly, then the buffer is not mutated.
//
// The prefix and indent strings must be composed of only spaces and/or tabs.
func (v *Value) Indent(prefix, indent string) error {
	return v.reformat(false, true, prefix, indent)
}

// Canonicalize canonicalizes the raw JSON value according to the
// JSON Canonicalization Scheme (JCS) as defined by RFC 8785
// where it produces a stable representation of a JSON value.
//
// The output stability is dependent on the stability of the application data
// (see RFC 8785, Appendix E). It cannot produce stable output from
// fundamentally unstable input. For example, if the JSON value
// contains ephemeral data (e.g., a frequently changing timestamp),
// then the value is still unstable regardless of whether this is called.
//
// Note that JCS treats all JSON numbers as IEEE 754 double precision numbers.
// Any numbers with precision beyond what is representable by that form
// will lose their precision when canonicalized. For example, integer values
// beyond ±2⁵³ will lose their precision. It is recommended that
// int64 and uint64 data types be represented as a JSON string.
//
// It is guaranteed to succeed if the input is valid.
// If the value is already canonicalized, then the buffer is not mutated.
func (v *Value) Canonicalize() error {
	return v.reformat(true, false, "", "")
}

// TODO: Instead of implementing the v1 Marshaler/Unmarshaler,
// consider implementing the v2 versions instead.

// MarshalJSON returns v as the JSON encoding of v.
// It returns the stored value as the raw JSON output without any validation.
// If v is nil, then this returns a JSON null.
func (v Value) MarshalJSON() ([]byte, error) {
	// NOTE: This matches the behavior of v1 json.RawMessage.MarshalJSON.
	if v == nil {
		return []byte("null"), nil
	}
	return v, nil
}

// UnmarshalJSON sets v as the JSON encoding of b.
// It stores a copy of the provided raw JSON input without any validation.
func (v *Value) UnmarshalJSON(b []byte) error {
	// NOTE: This matches the behavior of v1 json.RawMessage.UnmarshalJSON.
	if v == nil {
		return errors.New("json.Value: UnmarshalJSON on nil pointer")
	}
	*v = append((*v)[:0], b...)
	return nil
}

// Kind returns the starting token kind.
// For a valid value, this will never include '}' or ']'.
func (v Value) Kind() Kind {
	if v := v[jsonwire.ConsumeWhitespace(v):]; len(v) > 0 {
		return Kind(v[0]).normalize()
	}
	return invalidKind
}

func (v *Value) reformat(canonical, multiline bool, prefix, indent string) error {
	// Write the entire value to reformat all tokens and whitespace.
	e := getBufferedEncoder()
	defer putBufferedEncoder(e)
	eo := &e.s.Struct
	if canonical {
		eo.Flags.Set(jsonflags.AllowInvalidUTF8 | 0)    // per RFC 8785, section 3.2.4
		eo.Flags.Set(jsonflags.AllowDuplicateNames | 0) // per RFC 8785, section 3.1
		eo.Flags.Set(jsonflags.CanonicalizeNumbers | 1) // per RFC 8785, section 3.2.2.3
		eo.Flags.Set(jsonflags.PreserveRawStrings | 0)  // per RFC 8785, section 3.2.2.2
		eo.Flags.Set(jsonflags.EscapeForHTML | 0)       // per RFC 8785, section 3.2.2.2
		eo.Flags.Set(jsonflags.EscapeForJS | 0)         // per RFC 8785, section 3.2.2.2
		eo.Flags.Set(jsonflags.Expand | 0)              // per RFC 8785, section 3.2.1
	} else {
		if s := strings.TrimLeft(prefix, " \t"); len(s) > 0 {
			panic("json: invalid character " + jsonwire.QuoteRune(s) + " in indent prefix")
		}
		if s := strings.TrimLeft(indent, " \t"); len(s) > 0 {
			panic("json: invalid character " + jsonwire.QuoteRune(s) + " in indent")
		}
		eo.Flags.Set(jsonflags.AllowInvalidUTF8 | 1)
		eo.Flags.Set(jsonflags.AllowDuplicateNames | 1)
		eo.Flags.Set(jsonflags.PreserveRawStrings | 1)
		if multiline {
			eo.Flags.Set(jsonflags.Expand | 1)
			eo.Flags.Set(jsonflags.Indent | 1)
			eo.Flags.Set(jsonflags.IndentPrefix | 1)
			eo.IndentPrefix = prefix
			eo.Indent = indent
		} else {
			eo.Flags.Set(jsonflags.Expand | 0)
		}
	}
	eo.Flags.Set(jsonflags.OmitTopLevelNewline | 1)
	if err := e.s.WriteValue(*v); err != nil {
		return err
	}

	// For canonical output, we may need to reorder object members.
	if canonical {
		// Obtain a buffered encoder just to use its internal buffer as
		// a scratch buffer in reorderObjects for reordering object members.
		e2 := getBufferedEncoder()
		defer putBufferedEncoder(e2)

		// Disable redundant checks performed earlier during encoding.
		d := getBufferedDecoder(e.s.Buf)
		defer putBufferedDecoder(d)
		d.s.Flags.Set(jsonflags.AllowDuplicateNames | jsonflags.AllowInvalidUTF8 | 1)
		reorderObjects(d, &e2.s.Buf) // per RFC 8785, section 3.2.3
	}

	// Store the result back into the value if different.
	if !bytes.Equal(*v, e.s.Buf) {
		*v = append((*v)[:0], e.s.Buf...)
	}
	return nil
}

type memberName struct {
	// name is the unescaped name.
	name []byte
	// before and after are byte offsets into Decoder.buf that represents
	// the entire name/value pair. It may contain leading commas.
	before, after int64
}

var memberNamePool = sync.Pool{New: func() any { return new([]memberName) }}

func getMemberNames() *[]memberName {
	ns := memberNamePool.Get().(*[]memberName)
	*ns = (*ns)[:0]
	return ns
}
func putMemberNames(ns *[]memberName) {
	if cap(*ns) < 1<<10 {
		clear(*ns) // avoid pinning name
		memberNamePool.Put(ns)
	}
}

// reorderObjects recursively reorders all object members in place
// according to the ordering specified in RFC 8785, section 3.2.3.
//
// Pre-conditions:
//   - The value is valid (i.e., no decoder errors should ever occur).
//   - The value is compact (i.e., no whitespace is present).
//   - Initial call is provided a Decoder reading from the start of v.
//
// Post-conditions:
//   - Exactly one JSON value is read from the Decoder.
//   - All fully-parsed JSON objects are reordered by directly moving
//     the members in the value buffer.
//
// The runtime is approximately O(n·log(n)) + O(m·log(m)),
// where n is len(v) and m is the total number of object members.
func reorderObjects(d *Decoder, scratch *[]byte) {
	switch tok, _ := d.ReadToken(); tok.Kind() {
	case '{':
		// Iterate and collect the name and offsets for every object member.
		members := getMemberNames()
		defer putMemberNames(members)
		var prevName []byte
		isSorted := true

		beforeBody := d.InputOffset() // offset after '{'
		for d.PeekKind() != '}' {
			beforeName := d.InputOffset()
			var flags jsonwire.ValueFlags
			name, _ := d.s.ReadValue(&flags)
			name = jsonwire.UnquoteMayCopy(name, flags.IsVerbatim())
			reorderObjects(d, scratch)
			afterValue := d.InputOffset()

			if isSorted && len(*members) > 0 {
				isSorted = jsonwire.CompareUTF16(prevName, []byte(name)) < 0
			}
			*members = append(*members, memberName{name, beforeName, afterValue})
			prevName = name
		}
		afterBody := d.InputOffset() // offset before '}'
		d.ReadToken()

		// Sort the members; return early if it's already sorted.
		if isSorted {
			return
		}
		slices.SortFunc(*members, func(x, y memberName) int {
			return jsonwire.CompareUTF16(x.name, y.name)
		})

		// Append the reordered members to a new buffer,
		// then copy the reordered members back over the original members.
		// Avoid swapping in place since each member may be a different size
		// where moving a member over a smaller member may corrupt the data
		// for subsequent members before they have been moved.
		//
		// The following invariant must hold:
		//	sum([m.after-m.before for m in members]) == afterBody-beforeBody
		sorted := (*scratch)[:0]
		for i, member := range *members {
			if d.s.buf[member.before] == ',' {
				member.before++ // trim leading comma
			}
			sorted = append(sorted, d.s.buf[member.before:member.after]...)
			if i < len(*members)-1 {
				sorted = append(sorted, ',') // append trailing comma
			}
		}
		if int(afterBody-beforeBody) != len(sorted) {
			panic("BUG: length invariant violated")
		}
		copy(d.s.buf[beforeBody:afterBody], sorted)

		// Update scratch buffer to the largest amount ever used.
		if len(sorted) > len(*scratch) {
			*scratch = sorted
		}
	case '[':
		for d.PeekKind() != ']' {
			reorderObjects(d, scratch)
		}
		d.ReadToken()
	}
}
