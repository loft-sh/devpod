// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsontext

import (
	"bytes"
	"errors"
	"io"

	"github.com/go-json-experiment/json/internal/jsonflags"
	"github.com/go-json-experiment/json/internal/jsonopts"
	"github.com/go-json-experiment/json/internal/jsonwire"
)

// NOTE: The logic for decoding is complicated by the fact that reading from
// an io.Reader into a temporary buffer means that the buffer may contain a
// truncated portion of some valid input, requiring the need to fetch more data.
//
// This file is structured in the following way:
//
//   - consumeXXX functions parse an exact JSON token from a []byte.
//     If the buffer appears truncated, then it returns io.ErrUnexpectedEOF.
//     The consumeSimpleXXX functions are so named because they only handle
//     a subset of the grammar for the JSON token being parsed.
//     They do not handle the full grammar to keep these functions inlinable.
//
//   - Decoder.consumeXXX methods parse the next JSON token from Decoder.buf,
//     automatically fetching more input if necessary. These methods take
//     a position relative to the start of Decoder.buf as an argument and
//     return the end of the consumed JSON token as a position,
//     also relative to the start of Decoder.buf.
//
//   - In the event of an I/O errors or state machine violations,
//     the implementation avoids mutating the state of Decoder
//     (aside from the book-keeping needed to implement Decoder.fetch).
//     For this reason, only Decoder.ReadToken and Decoder.ReadValue are
//     responsible for updated Decoder.prevStart and Decoder.prevEnd.
//
//   - For performance, much of the implementation uses the pattern of calling
//     the inlinable consumeXXX functions first, and if more work is necessary,
//     then it calls the slower Decoder.consumeXXX methods.
//     TODO: Revisit this pattern if the Go compiler provides finer control
//     over exactly which calls are inlined or not.

// Decoder is a streaming decoder for raw JSON tokens and values.
// It is used to read a stream of top-level JSON values,
// each separated by optional whitespace characters.
//
// [Decoder.ReadToken] and [Decoder.ReadValue] calls may be interleaved.
// For example, the following JSON value:
//
//	{"name":"value","array":[null,false,true,3.14159],"object":{"k":"v"}}
//
// can be parsed with the following calls (ignoring errors for brevity):
//
//	d.ReadToken() // {
//	d.ReadToken() // "name"
//	d.ReadToken() // "value"
//	d.ReadValue() // "array"
//	d.ReadToken() // [
//	d.ReadToken() // null
//	d.ReadToken() // false
//	d.ReadValue() // true
//	d.ReadToken() // 3.14159
//	d.ReadToken() // ]
//	d.ReadValue() // "object"
//	d.ReadValue() // {"k":"v"}
//	d.ReadToken() // }
//
// The above is one of many possible sequence of calls and
// may not represent the most sensible method to call for any given token/value.
// For example, it is probably more common to call [Decoder.ReadToken] to obtain a
// string token for object names.
type Decoder struct {
	s decoderState
}

// decoderState is the low-level state of Decoder.
// It has exported fields and method for use by the "json" package.
type decoderState struct {
	state
	decodeBuffer
	jsonopts.Struct

	StringCache *[256]string // only used when unmarshaling; identical to json.stringCache
}

// decodeBuffer is a buffer split into 4 segments:
//
//   - buf[0:prevEnd]         // already read portion of the buffer
//   - buf[prevStart:prevEnd] // previously read value
//   - buf[prevEnd:len(buf)]  // unread portion of the buffer
//   - buf[len(buf):cap(buf)] // unused portion of the buffer
//
// Invariants:
//
//	0 ≤ prevStart ≤ prevEnd ≤ len(buf) ≤ cap(buf)
type decodeBuffer struct {
	peekPos int   // non-zero if valid offset into buf for start of next token
	peekErr error // implies peekPos is -1

	buf       []byte // may alias rd if it is a bytes.Buffer
	prevStart int
	prevEnd   int

	// baseOffset is added to prevStart and prevEnd to obtain
	// the absolute offset relative to the start of io.Reader stream.
	baseOffset int64

	rd io.Reader
}

// NewDecoder constructs a new streaming decoder reading from r.
//
// If r is a [bytes.Buffer], then the decoder parses directly from the buffer
// without first copying the contents to an intermediate buffer.
// Additional writes to the buffer must not occur while the decoder is in use.
func NewDecoder(r io.Reader, opts ...Options) *Decoder {
	d := new(Decoder)
	d.Reset(r, opts...)
	return d
}

// Reset resets a decoder such that it is reading afresh from r and
// configured with the provided options. Reset must not be called on an
// a Decoder passed to the [encoding/json/v2.UnmarshalerV2.UnmarshalJSONV2] method
// or the [encoding/json/v2.UnmarshalFuncV2] function.
func (d *Decoder) Reset(r io.Reader, opts ...Options) {
	switch {
	case d == nil:
		panic("jsontext: invalid nil Decoder")
	case r == nil:
		panic("jsontext: invalid nil io.Writer")
	case d.s.Flags.Get(jsonflags.WithinArshalCall):
		panic("jsontext: cannot reset Decoder passed to json.UnmarshalerV2")
	}
	d.s.reset(nil, r, opts...)
}

func (d *decoderState) reset(b []byte, r io.Reader, opts ...Options) {
	d.state.reset()
	d.decodeBuffer = decodeBuffer{buf: b, rd: r}
	d.Struct = jsonopts.Struct{}
	d.Struct.Join(opts...)
}

var errBufferWriteAfterNext = errors.New("invalid bytes.Buffer.Write call after calling bytes.Buffer.Next")

// fetch reads at least 1 byte from the underlying io.Reader.
// It returns io.ErrUnexpectedEOF if zero bytes were read and io.EOF was seen.
func (d *decoderState) fetch() error {
	if d.rd == nil {
		return io.ErrUnexpectedEOF
	}

	// Inform objectNameStack that we are about to fetch new buffer content.
	d.Names.copyQuotedBuffer(d.buf)

	// Specialize bytes.Buffer for better performance.
	if bb, ok := d.rd.(*bytes.Buffer); ok {
		switch {
		case bb.Len() == 0:
			return io.ErrUnexpectedEOF
		case len(d.buf) == 0:
			d.buf = bb.Next(bb.Len()) // "read" all data in the buffer
			return nil
		default:
			// This only occurs if a partially filled bytes.Buffer was provided
			// and more data is written to it while Decoder is reading from it.
			// This practice will lead to data corruption since future writes
			// may overwrite the contents of the current buffer.
			//
			// The user is trying to use a bytes.Buffer as a pipe,
			// but a bytes.Buffer is poor implementation of a pipe,
			// the purpose-built io.Pipe should be used instead.
			return &ioError{action: "read", err: errBufferWriteAfterNext}
		}
	}

	// Allocate initial buffer if empty.
	if cap(d.buf) == 0 {
		d.buf = make([]byte, 0, 64)
	}

	// Check whether to grow the buffer.
	const maxBufferSize = 4 << 10
	const growthSizeFactor = 2 // higher value is faster
	const growthRateFactor = 2 // higher value is slower
	// By default, grow if below the maximum buffer size.
	grow := cap(d.buf) <= maxBufferSize/growthSizeFactor
	// Growing can be expensive, so only grow
	// if a sufficient number of bytes have been processed.
	grow = grow && int64(cap(d.buf)) < d.previousOffsetEnd()/growthRateFactor
	// If prevStart==0, then fetch was called in order to fetch more data
	// to finish consuming a large JSON value contiguously.
	// Grow if less than 25% of the remaining capacity is available.
	// Note that this may cause the input buffer to exceed maxBufferSize.
	grow = grow || (d.prevStart == 0 && len(d.buf) >= 3*cap(d.buf)/4)

	if grow {
		// Allocate a new buffer and copy the contents of the old buffer over.
		// TODO: Provide a hard limit on the maximum internal buffer size?
		buf := make([]byte, 0, cap(d.buf)*growthSizeFactor)
		d.buf = append(buf, d.buf[d.prevStart:]...)
	} else {
		// Move unread portion of the data to the front.
		n := copy(d.buf[:cap(d.buf)], d.buf[d.prevStart:])
		d.buf = d.buf[:n]
	}
	d.baseOffset += int64(d.prevStart)
	d.prevEnd -= d.prevStart
	d.prevStart = 0

	// Read more data into the internal buffer.
	for {
		n, err := d.rd.Read(d.buf[len(d.buf):cap(d.buf)])
		switch {
		case n > 0:
			d.buf = d.buf[:len(d.buf)+n]
			return nil // ignore errors if any bytes are read
		case err == io.EOF:
			return io.ErrUnexpectedEOF
		case err != nil:
			return &ioError{action: "read", err: err}
		default:
			continue // Read returned (0, nil)
		}
	}
}

const invalidateBufferByte = '#' // invalid starting character for JSON grammar

// invalidatePreviousRead invalidates buffers returned by Peek and Read calls
// so that the first byte is an invalid character.
// This Hyrum-proofs the API against faulty application code that assumes
// values returned by ReadValue remain valid past subsequent Read calls.
func (d *decodeBuffer) invalidatePreviousRead() {
	// Avoid mutating the buffer if d.rd is nil which implies that d.buf
	// is provided by the user code and may not expect mutations.
	isBytesBuffer := func(r io.Reader) bool {
		_, ok := r.(*bytes.Buffer)
		return ok
	}
	if d.rd != nil && !isBytesBuffer(d.rd) && d.prevStart < d.prevEnd && uint(d.prevStart) < uint(len(d.buf)) {
		d.buf[d.prevStart] = invalidateBufferByte
		d.prevStart = d.prevEnd
	}
}

// needMore reports whether there are no more unread bytes.
func (d *decodeBuffer) needMore(pos int) bool {
	// NOTE: The arguments and logic are kept simple to keep this inlinable.
	return pos == len(d.buf)
}

// injectSyntacticErrorWithPosition wraps a SyntacticError with the position,
// otherwise it returns the error as is.
// It takes a position relative to the start of the start of d.buf.
func (d *decodeBuffer) injectSyntacticErrorWithPosition(err error, pos int) error {
	if serr, ok := err.(*SyntacticError); ok {
		return serr.withOffset(d.baseOffset + int64(pos))
	}
	return err
}

func (d *decodeBuffer) previousOffsetStart() int64 { return d.baseOffset + int64(d.prevStart) }
func (d *decodeBuffer) previousOffsetEnd() int64   { return d.baseOffset + int64(d.prevEnd) }
func (d *decodeBuffer) PreviousBuffer() []byte     { return d.buf[d.prevStart:d.prevEnd] }
func (d *decodeBuffer) unreadBuffer() []byte       { return d.buf[d.prevEnd:len(d.buf)] }

// PeekKind retrieves the next token kind, but does not advance the read offset.
// It returns 0 if there are no more tokens.
func (d *Decoder) PeekKind() Kind {
	return d.s.PeekKind()
}
func (d *decoderState) PeekKind() Kind {
	// Check whether we have a cached peek result.
	if d.peekPos > 0 {
		return Kind(d.buf[d.peekPos]).normalize()
	}

	var err error
	d.invalidatePreviousRead()
	pos := d.prevEnd

	// Consume leading whitespace.
	pos += jsonwire.ConsumeWhitespace(d.buf[pos:])
	if d.needMore(pos) {
		if pos, err = d.consumeWhitespace(pos); err != nil {
			if err == io.ErrUnexpectedEOF && d.Tokens.Depth() == 1 {
				err = io.EOF // EOF possibly if no Tokens present after top-level value
			}
			d.peekPos, d.peekErr = -1, err
			return invalidKind
		}
	}

	// Consume colon or comma.
	var delim byte
	if c := d.buf[pos]; c == ':' || c == ',' {
		delim = c
		pos += 1
		pos += jsonwire.ConsumeWhitespace(d.buf[pos:])
		if d.needMore(pos) {
			if pos, err = d.consumeWhitespace(pos); err != nil {
				d.peekPos, d.peekErr = -1, d.checkDelimBeforeIOError(delim, err)
				return invalidKind
			}
		}
	}
	next := Kind(d.buf[pos]).normalize()
	if d.Tokens.needDelim(next) != delim {
		d.peekPos, d.peekErr = -1, d.checkDelim(delim, next)
		return invalidKind
	}

	// This may set peekPos to zero, which is indistinguishable from
	// the uninitialized state. While a small hit to performance, it is correct
	// since ReadValue and ReadToken will disregard the cached result and
	// recompute the next kind.
	d.peekPos, d.peekErr = pos, nil
	return next
}

// checkDelimBeforeIOError checks whether the delim is even valid
// before returning an IO error, which occurs after the delim.
func (d *decoderState) checkDelimBeforeIOError(delim byte, err error) error {
	// Since an IO error occurred, we do not know what the next kind is.
	// However, knowing the next kind is necessary to validate
	// whether the current delim is at least potentially valid.
	// Since a JSON string is always valid as the next token,
	// conservatively assume that is the next kind for validation.
	const next = Kind('"')
	if d.Tokens.needDelim(next) != delim {
		err = d.checkDelim(delim, next)
	}
	return err
}

// checkDelim checks whether delim is valid for the given next kind.
func (d *decoderState) checkDelim(delim byte, next Kind) error {
	pos := d.prevEnd // restore position to right after leading whitespace
	pos += jsonwire.ConsumeWhitespace(d.buf[pos:])
	err := d.Tokens.checkDelim(delim, next)
	return d.injectSyntacticErrorWithPosition(err, pos)
}

// SkipValue is semantically equivalent to calling [Decoder.ReadValue] and discarding
// the result except that memory is not wasted trying to hold the entire result.
func (d *Decoder) SkipValue() error {
	return d.s.SkipValue()
}
func (d *decoderState) SkipValue() error {
	switch d.PeekKind() {
	case '{', '[':
		// For JSON objects and arrays, keep skipping all tokens
		// until the depth matches the starting depth.
		depth := d.Tokens.Depth()
		for {
			if _, err := d.ReadToken(); err != nil {
				return err
			}
			if depth >= d.Tokens.Depth() {
				return nil
			}
		}
	default:
		// Trying to skip a value when the next token is a '}' or ']'
		// will result in an error being returned here.
		var flags jsonwire.ValueFlags
		if _, err := d.ReadValue(&flags); err != nil {
			return err
		}
		return nil
	}
}

// ReadToken reads the next [Token], advancing the read offset.
// The returned token is only valid until the next Peek, Read, or Skip call.
// It returns [io.EOF] if there are no more tokens.
func (d *Decoder) ReadToken() (Token, error) {
	return d.s.ReadToken()
}
func (d *decoderState) ReadToken() (Token, error) {
	// Determine the next kind.
	var err error
	var next Kind
	pos := d.peekPos
	if pos != 0 {
		// Use cached peek result.
		if d.peekErr != nil {
			err := d.peekErr
			d.peekPos, d.peekErr = 0, nil // possibly a transient I/O error
			return Token{}, err
		}
		next = Kind(d.buf[pos]).normalize()
		d.peekPos = 0 // reset cache
	} else {
		d.invalidatePreviousRead()
		pos = d.prevEnd

		// Consume leading whitespace.
		pos += jsonwire.ConsumeWhitespace(d.buf[pos:])
		if d.needMore(pos) {
			if pos, err = d.consumeWhitespace(pos); err != nil {
				if err == io.ErrUnexpectedEOF && d.Tokens.Depth() == 1 {
					err = io.EOF // EOF possibly if no Tokens present after top-level value
				}
				return Token{}, err
			}
		}

		// Consume colon or comma.
		var delim byte
		if c := d.buf[pos]; c == ':' || c == ',' {
			delim = c
			pos += 1
			pos += jsonwire.ConsumeWhitespace(d.buf[pos:])
			if d.needMore(pos) {
				if pos, err = d.consumeWhitespace(pos); err != nil {
					return Token{}, d.checkDelimBeforeIOError(delim, err)
				}
			}
		}
		next = Kind(d.buf[pos]).normalize()
		if d.Tokens.needDelim(next) != delim {
			return Token{}, d.checkDelim(delim, next)
		}
	}

	// Handle the next token.
	var n int
	switch next {
	case 'n':
		if jsonwire.ConsumeNull(d.buf[pos:]) == 0 {
			pos, err = d.consumeLiteral(pos, "null")
			if err != nil {
				return Token{}, d.injectSyntacticErrorWithPosition(err, pos)
			}
		} else {
			pos += len("null")
		}
		if err = d.Tokens.appendLiteral(); err != nil {
			return Token{}, d.injectSyntacticErrorWithPosition(err, pos-len("null")) // report position at start of literal
		}
		d.prevStart, d.prevEnd = pos, pos
		return Null, nil

	case 'f':
		if jsonwire.ConsumeFalse(d.buf[pos:]) == 0 {
			pos, err = d.consumeLiteral(pos, "false")
			if err != nil {
				return Token{}, d.injectSyntacticErrorWithPosition(err, pos)
			}
		} else {
			pos += len("false")
		}
		if err = d.Tokens.appendLiteral(); err != nil {
			return Token{}, d.injectSyntacticErrorWithPosition(err, pos-len("false")) // report position at start of literal
		}
		d.prevStart, d.prevEnd = pos, pos
		return False, nil

	case 't':
		if jsonwire.ConsumeTrue(d.buf[pos:]) == 0 {
			pos, err = d.consumeLiteral(pos, "true")
			if err != nil {
				return Token{}, d.injectSyntacticErrorWithPosition(err, pos)
			}
		} else {
			pos += len("true")
		}
		if err = d.Tokens.appendLiteral(); err != nil {
			return Token{}, d.injectSyntacticErrorWithPosition(err, pos-len("true")) // report position at start of literal
		}
		d.prevStart, d.prevEnd = pos, pos
		return True, nil

	case '"':
		var flags jsonwire.ValueFlags // TODO: Preserve this in Token?
		if n = jsonwire.ConsumeSimpleString(d.buf[pos:]); n == 0 {
			oldAbsPos := d.baseOffset + int64(pos)
			pos, err = d.consumeString(&flags, pos)
			newAbsPos := d.baseOffset + int64(pos)
			n = int(newAbsPos - oldAbsPos)
			if err != nil {
				return Token{}, d.injectSyntacticErrorWithPosition(err, pos)
			}
		} else {
			pos += n
		}
		if !d.Flags.Get(jsonflags.AllowDuplicateNames) && d.Tokens.Last.NeedObjectName() {
			if !d.Tokens.Last.isValidNamespace() {
				return Token{}, errInvalidNamespace
			}
			if d.Tokens.Last.isActiveNamespace() && !d.Namespaces.Last().insertQuoted(d.buf[pos-n:pos], flags.IsVerbatim()) {
				err = newDuplicateNameError(d.buf[pos-n : pos])
				return Token{}, d.injectSyntacticErrorWithPosition(err, pos-n) // report position at start of string
			}
			d.Names.ReplaceLastQuotedOffset(pos - n) // only replace if insertQuoted succeeds
		}
		if err = d.Tokens.appendString(); err != nil {
			return Token{}, d.injectSyntacticErrorWithPosition(err, pos-n) // report position at start of string
		}
		d.prevStart, d.prevEnd = pos-n, pos
		return Token{raw: &d.decodeBuffer, num: uint64(d.previousOffsetStart())}, nil

	case '0':
		// NOTE: Since JSON numbers are not self-terminating,
		// we need to make sure that the next byte is not part of a number.
		if n = jsonwire.ConsumeSimpleNumber(d.buf[pos:]); n == 0 || d.needMore(pos+n) {
			oldAbsPos := d.baseOffset + int64(pos)
			pos, err = d.consumeNumber(pos)
			newAbsPos := d.baseOffset + int64(pos)
			n = int(newAbsPos - oldAbsPos)
			if err != nil {
				return Token{}, d.injectSyntacticErrorWithPosition(err, pos)
			}
		} else {
			pos += n
		}
		if err = d.Tokens.appendNumber(); err != nil {
			return Token{}, d.injectSyntacticErrorWithPosition(err, pos-n) // report position at start of number
		}
		d.prevStart, d.prevEnd = pos-n, pos
		return Token{raw: &d.decodeBuffer, num: uint64(d.previousOffsetStart())}, nil

	case '{':
		if err = d.Tokens.pushObject(); err != nil {
			return Token{}, d.injectSyntacticErrorWithPosition(err, pos)
		}
		if !d.Flags.Get(jsonflags.AllowDuplicateNames) {
			d.Names.push()
			d.Namespaces.push()
		}
		pos += 1
		d.prevStart, d.prevEnd = pos, pos
		return ObjectStart, nil

	case '}':
		if err = d.Tokens.popObject(); err != nil {
			return Token{}, d.injectSyntacticErrorWithPosition(err, pos)
		}
		if !d.Flags.Get(jsonflags.AllowDuplicateNames) {
			d.Names.pop()
			d.Namespaces.pop()
		}
		pos += 1
		d.prevStart, d.prevEnd = pos, pos
		return ObjectEnd, nil

	case '[':
		if err = d.Tokens.pushArray(); err != nil {
			return Token{}, d.injectSyntacticErrorWithPosition(err, pos)
		}
		pos += 1
		d.prevStart, d.prevEnd = pos, pos
		return ArrayStart, nil

	case ']':
		if err = d.Tokens.popArray(); err != nil {
			return Token{}, d.injectSyntacticErrorWithPosition(err, pos)
		}
		pos += 1
		d.prevStart, d.prevEnd = pos, pos
		return ArrayEnd, nil

	default:
		err = newInvalidCharacterError(d.buf[pos:], "at start of token")
		return Token{}, d.injectSyntacticErrorWithPosition(err, pos)
	}
}

// ReadValue returns the next raw JSON value, advancing the read offset.
// The value is stripped of any leading or trailing whitespace and
// contains the exact bytes of the input, which may contain invalid UTF-8
// if [AllowInvalidUTF8] is specified.
//
// The returned value is only valid until the next Peek, Read, or Skip call and
// may not be mutated while the Decoder remains in use.
// If the decoder is currently at the end token for an object or array,
// then it reports a [SyntacticError] and the internal state remains unchanged.
// It returns [io.EOF] if there are no more values.
func (d *Decoder) ReadValue() (Value, error) {
	var flags jsonwire.ValueFlags
	return d.s.ReadValue(&flags)
}
func (d *decoderState) ReadValue(flags *jsonwire.ValueFlags) (Value, error) {
	// Determine the next kind.
	var err error
	var next Kind
	pos := d.peekPos
	if pos != 0 {
		// Use cached peek result.
		if d.peekErr != nil {
			err := d.peekErr
			d.peekPos, d.peekErr = 0, nil // possibly a transient I/O error
			return nil, err
		}
		next = Kind(d.buf[pos]).normalize()
		d.peekPos = 0 // reset cache
	} else {
		d.invalidatePreviousRead()
		pos = d.prevEnd

		// Consume leading whitespace.
		pos += jsonwire.ConsumeWhitespace(d.buf[pos:])
		if d.needMore(pos) {
			if pos, err = d.consumeWhitespace(pos); err != nil {
				if err == io.ErrUnexpectedEOF && d.Tokens.Depth() == 1 {
					err = io.EOF // EOF possibly if no Tokens present after top-level value
				}
				return nil, err
			}
		}

		// Consume colon or comma.
		var delim byte
		if c := d.buf[pos]; c == ':' || c == ',' {
			delim = c
			pos += 1
			pos += jsonwire.ConsumeWhitespace(d.buf[pos:])
			if d.needMore(pos) {
				if pos, err = d.consumeWhitespace(pos); err != nil {
					return nil, d.checkDelimBeforeIOError(delim, err)
				}
			}
		}
		next = Kind(d.buf[pos]).normalize()
		if d.Tokens.needDelim(next) != delim {
			return nil, d.checkDelim(delim, next)
		}
	}

	// Handle the next value.
	oldAbsPos := d.baseOffset + int64(pos)
	pos, err = d.consumeValue(flags, pos, d.Tokens.Depth())
	newAbsPos := d.baseOffset + int64(pos)
	n := int(newAbsPos - oldAbsPos)
	if err != nil {
		return nil, d.injectSyntacticErrorWithPosition(err, pos)
	}
	switch next {
	case 'n', 't', 'f':
		err = d.Tokens.appendLiteral()
	case '"':
		if !d.Flags.Get(jsonflags.AllowDuplicateNames) && d.Tokens.Last.NeedObjectName() {
			if !d.Tokens.Last.isValidNamespace() {
				err = errInvalidNamespace
				break
			}
			if d.Tokens.Last.isActiveNamespace() && !d.Namespaces.Last().insertQuoted(d.buf[pos-n:pos], flags.IsVerbatim()) {
				err = newDuplicateNameError(d.buf[pos-n : pos])
				break
			}
			d.Names.ReplaceLastQuotedOffset(pos - n) // only replace if insertQuoted succeeds
		}
		err = d.Tokens.appendString()
	case '0':
		err = d.Tokens.appendNumber()
	case '{':
		if err = d.Tokens.pushObject(); err != nil {
			break
		}
		if err = d.Tokens.popObject(); err != nil {
			panic("BUG: popObject should never fail immediately after pushObject: " + err.Error())
		}
	case '[':
		if err = d.Tokens.pushArray(); err != nil {
			break
		}
		if err = d.Tokens.popArray(); err != nil {
			panic("BUG: popArray should never fail immediately after pushArray: " + err.Error())
		}
	}
	if err != nil {
		return nil, d.injectSyntacticErrorWithPosition(err, pos-n) // report position at start of value
	}
	d.prevEnd = pos
	d.prevStart = pos - n
	return d.buf[pos-n : pos : pos], nil
}

// CheckEOF verifies that the input has no more data.
func (d *decoderState) CheckEOF() error {
	switch pos, err := d.consumeWhitespace(d.prevEnd); err {
	case nil:
		err := newInvalidCharacterError(d.buf[pos:], "after top-level value")
		return d.injectSyntacticErrorWithPosition(err, pos)
	case io.ErrUnexpectedEOF:
		return nil
	default:
		return err
	}
}

// consumeWhitespace consumes all whitespace starting at d.buf[pos:].
// It returns the new position in d.buf immediately after the last whitespace.
// If it returns nil, there is guaranteed to at least be one unread byte.
//
// The following pattern is common in this implementation:
//
//	pos += jsonwire.ConsumeWhitespace(d.buf[pos:])
//	if d.needMore(pos) {
//		if pos, err = d.consumeWhitespace(pos); err != nil {
//			return ...
//		}
//	}
//
// It is difficult to simplify this without sacrificing performance since
// consumeWhitespace must be inlined. The body of the if statement is
// executed only in rare situations where we need to fetch more data.
// Since fetching may return an error, we also need to check the error.
func (d *decoderState) consumeWhitespace(pos int) (newPos int, err error) {
	for {
		pos += jsonwire.ConsumeWhitespace(d.buf[pos:])
		if d.needMore(pos) {
			absPos := d.baseOffset + int64(pos)
			err = d.fetch() // will mutate d.buf and invalidate pos
			pos = int(absPos - d.baseOffset)
			if err != nil {
				return pos, err
			}
			continue
		}
		return pos, nil
	}
}

// consumeValue consumes a single JSON value starting at d.buf[pos:].
// It returns the new position in d.buf immediately after the value.
func (d *decoderState) consumeValue(flags *jsonwire.ValueFlags, pos, depth int) (newPos int, err error) {
	for {
		var n int
		var err error
		switch next := Kind(d.buf[pos]).normalize(); next {
		case 'n':
			if n = jsonwire.ConsumeNull(d.buf[pos:]); n == 0 {
				n, err = jsonwire.ConsumeLiteral(d.buf[pos:], "null")
			}
		case 'f':
			if n = jsonwire.ConsumeFalse(d.buf[pos:]); n == 0 {
				n, err = jsonwire.ConsumeLiteral(d.buf[pos:], "false")
			}
		case 't':
			if n = jsonwire.ConsumeTrue(d.buf[pos:]); n == 0 {
				n, err = jsonwire.ConsumeLiteral(d.buf[pos:], "true")
			}
		case '"':
			if n = jsonwire.ConsumeSimpleString(d.buf[pos:]); n == 0 {
				return d.consumeString(flags, pos)
			}
		case '0':
			// NOTE: Since JSON numbers are not self-terminating,
			// we need to make sure that the next byte is not part of a number.
			if n = jsonwire.ConsumeSimpleNumber(d.buf[pos:]); n == 0 || d.needMore(pos+n) {
				return d.consumeNumber(pos)
			}
		case '{':
			return d.consumeObject(flags, pos, depth)
		case '[':
			return d.consumeArray(flags, pos, depth)
		default:
			return pos, newInvalidCharacterError(d.buf[pos:], "at start of value")
		}
		if err == io.ErrUnexpectedEOF {
			absPos := d.baseOffset + int64(pos)
			err = d.fetch() // will mutate d.buf and invalidate pos
			pos = int(absPos - d.baseOffset)
			if err != nil {
				return pos, err
			}
			continue
		}
		return pos + n, err
	}
}

// consumeLiteral consumes a single JSON literal starting at d.buf[pos:].
// It returns the new position in d.buf immediately after the literal.
func (d *decoderState) consumeLiteral(pos int, lit string) (newPos int, err error) {
	for {
		n, err := jsonwire.ConsumeLiteral(d.buf[pos:], lit)
		if err == io.ErrUnexpectedEOF {
			absPos := d.baseOffset + int64(pos)
			err = d.fetch() // will mutate d.buf and invalidate pos
			pos = int(absPos - d.baseOffset)
			if err != nil {
				return pos, err
			}
			continue
		}
		return pos + n, err
	}
}

// consumeString consumes a single JSON string starting at d.buf[pos:].
// It returns the new position in d.buf immediately after the string.
func (d *decoderState) consumeString(flags *jsonwire.ValueFlags, pos int) (newPos int, err error) {
	var n int
	for {
		n, err = jsonwire.ConsumeStringResumable(flags, d.buf[pos:], n, !d.Flags.Get(jsonflags.AllowInvalidUTF8))
		if err == io.ErrUnexpectedEOF {
			absPos := d.baseOffset + int64(pos)
			err = d.fetch() // will mutate d.buf and invalidate pos
			pos = int(absPos - d.baseOffset)
			if err != nil {
				return pos, err
			}
			continue
		}
		return pos + n, err
	}
}

// consumeNumber consumes a single JSON number starting at d.buf[pos:].
// It returns the new position in d.buf immediately after the number.
func (d *decoderState) consumeNumber(pos int) (newPos int, err error) {
	var n int
	var state jsonwire.ConsumeNumberState
	for {
		n, state, err = jsonwire.ConsumeNumberResumable(d.buf[pos:], n, state)
		// NOTE: Since JSON numbers are not self-terminating,
		// we need to make sure that the next byte is not part of a number.
		if err == io.ErrUnexpectedEOF || d.needMore(pos+n) {
			mayTerminate := err == nil
			absPos := d.baseOffset + int64(pos)
			err = d.fetch() // will mutate d.buf and invalidate pos
			pos = int(absPos - d.baseOffset)
			if err != nil {
				if mayTerminate && err == io.ErrUnexpectedEOF {
					return pos + n, nil
				}
				return pos, err
			}
			continue
		}
		return pos + n, err
	}
}

// consumeObject consumes a single JSON object starting at d.buf[pos:].
// It returns the new position in d.buf immediately after the object.
func (d *decoderState) consumeObject(flags *jsonwire.ValueFlags, pos, depth int) (newPos int, err error) {
	var n int
	var names *objectNamespace
	if !d.Flags.Get(jsonflags.AllowDuplicateNames) {
		d.Namespaces.push()
		defer d.Namespaces.pop()
		names = d.Namespaces.Last()
	}

	// Handle before start.
	if uint(pos) >= uint(len(d.buf)) || d.buf[pos] != '{' {
		panic("BUG: consumeObject must be called with a buffer that starts with '{'")
	} else if depth == maxNestingDepth+1 {
		return pos, errMaxDepth
	}
	pos++

	// Handle after start.
	pos += jsonwire.ConsumeWhitespace(d.buf[pos:])
	if d.needMore(pos) {
		if pos, err = d.consumeWhitespace(pos); err != nil {
			return pos, err
		}
	}
	if d.buf[pos] == '}' {
		pos++
		return pos, nil
	}

	depth++
	for {
		// Handle before name.
		pos += jsonwire.ConsumeWhitespace(d.buf[pos:])
		if d.needMore(pos) {
			if pos, err = d.consumeWhitespace(pos); err != nil {
				return pos, err
			}
		}
		var flags2 jsonwire.ValueFlags
		if n = jsonwire.ConsumeSimpleString(d.buf[pos:]); n == 0 {
			oldAbsPos := d.baseOffset + int64(pos)
			pos, err = d.consumeString(&flags2, pos)
			newAbsPos := d.baseOffset + int64(pos)
			n = int(newAbsPos - oldAbsPos)
			flags.Join(flags2)
			if err != nil {
				return pos, err
			}
		} else {
			pos += n
		}
		if !d.Flags.Get(jsonflags.AllowDuplicateNames) && !names.insertQuoted(d.buf[pos-n:pos], flags2.IsVerbatim()) {
			return pos - n, newDuplicateNameError(d.buf[pos-n : pos])
		}

		// Handle after name.
		pos += jsonwire.ConsumeWhitespace(d.buf[pos:])
		if d.needMore(pos) {
			if pos, err = d.consumeWhitespace(pos); err != nil {
				return pos, err
			}
		}
		if d.buf[pos] != ':' {
			return pos, newInvalidCharacterError(d.buf[pos:], "after object name (expecting ':')")
		}
		pos++

		// Handle before value.
		pos += jsonwire.ConsumeWhitespace(d.buf[pos:])
		if d.needMore(pos) {
			if pos, err = d.consumeWhitespace(pos); err != nil {
				return pos, err
			}
		}
		pos, err = d.consumeValue(flags, pos, depth)
		if err != nil {
			return pos, err
		}

		// Handle after value.
		pos += jsonwire.ConsumeWhitespace(d.buf[pos:])
		if d.needMore(pos) {
			if pos, err = d.consumeWhitespace(pos); err != nil {
				return pos, err
			}
		}
		switch d.buf[pos] {
		case ',':
			pos++
			continue
		case '}':
			pos++
			return pos, nil
		default:
			return pos, newInvalidCharacterError(d.buf[pos:], "after object value (expecting ',' or '}')")
		}
	}
}

// consumeArray consumes a single JSON array starting at d.buf[pos:].
// It returns the new position in d.buf immediately after the array.
func (d *decoderState) consumeArray(flags *jsonwire.ValueFlags, pos, depth int) (newPos int, err error) {
	// Handle before start.
	if uint(pos) >= uint(len(d.buf)) || d.buf[pos] != '[' {
		panic("BUG: consumeArray must be called with a buffer that starts with '['")
	} else if depth == maxNestingDepth+1 {
		return pos, errMaxDepth
	}
	pos++

	// Handle after start.
	pos += jsonwire.ConsumeWhitespace(d.buf[pos:])
	if d.needMore(pos) {
		if pos, err = d.consumeWhitespace(pos); err != nil {
			return pos, err
		}
	}
	if d.buf[pos] == ']' {
		pos++
		return pos, nil
	}

	depth++
	for {
		// Handle before value.
		pos += jsonwire.ConsumeWhitespace(d.buf[pos:])
		if d.needMore(pos) {
			if pos, err = d.consumeWhitespace(pos); err != nil {
				return pos, err
			}
		}
		pos, err = d.consumeValue(flags, pos, depth)
		if err != nil {
			return pos, err
		}

		// Handle after value.
		pos += jsonwire.ConsumeWhitespace(d.buf[pos:])
		if d.needMore(pos) {
			if pos, err = d.consumeWhitespace(pos); err != nil {
				return pos, err
			}
		}
		switch d.buf[pos] {
		case ',':
			pos++
			continue
		case ']':
			pos++
			return pos, nil
		default:
			return pos, newInvalidCharacterError(d.buf[pos:], "after array value (expecting ',' or ']')")
		}
	}
}

// InputOffset returns the current input byte offset. It gives the location
// of the next byte immediately after the most recently returned token or value.
// The number of bytes actually read from the underlying [io.Reader] may be more
// than this offset due to internal buffering effects.
func (d *Decoder) InputOffset() int64 {
	return d.s.previousOffsetEnd()
}

// UnreadBuffer returns the data remaining in the unread buffer,
// which may contain zero or more bytes.
// The returned buffer must not be mutated while Decoder continues to be used.
// The buffer contents are valid until the next Peek, Read, or Skip call.
func (d *Decoder) UnreadBuffer() []byte {
	return d.s.unreadBuffer()
}

// StackDepth returns the depth of the state machine for read JSON data.
// Each level on the stack represents a nested JSON object or array.
// It is incremented whenever an [ObjectStart] or [ArrayStart] token is encountered
// and decremented whenever an [ObjectEnd] or [ArrayEnd] token is encountered.
// The depth is zero-indexed, where zero represents the top-level JSON value.
func (d *Decoder) StackDepth() int {
	// NOTE: Keep in sync with Encoder.StackDepth.
	return d.s.Tokens.Depth() - 1
}

// StackIndex returns information about the specified stack level.
// It must be a number between 0 and [Decoder.StackDepth], inclusive.
// For each level, it reports the kind:
//
//   - 0 for a level of zero,
//   - '{' for a level representing a JSON object, and
//   - '[' for a level representing a JSON array.
//
// It also reports the length of that JSON object or array.
// Each name and value in a JSON object is counted separately,
// so the effective number of members would be half the length.
// A complete JSON object must have an even length.
func (d *Decoder) StackIndex(i int) (Kind, int) {
	// NOTE: Keep in sync with Encoder.StackIndex.
	switch s := d.s.Tokens.index(i); {
	case i > 0 && s.isObject():
		return '{', s.Length()
	case i > 0 && s.isArray():
		return '[', s.Length()
	default:
		return 0, s.Length()
	}
}

// StackPointer returns a JSON Pointer (RFC 6901) to the most recently read value.
// Object names are only present if [AllowDuplicateNames] is false, otherwise
// object members are represented using their index within the object.
func (d *Decoder) StackPointer() string {
	d.s.Names.copyQuotedBuffer(d.s.buf)
	return string(d.s.appendStackPointer(nil))
}
