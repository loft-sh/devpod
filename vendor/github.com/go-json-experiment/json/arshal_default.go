// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"bytes"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"sync"

	"github.com/go-json-experiment/json/internal/jsonflags"
	"github.com/go-json-experiment/json/internal/jsonopts"
	"github.com/go-json-experiment/json/internal/jsonwire"
	"github.com/go-json-experiment/json/jsontext"
)

// optimizeCommon specifies whether to use optimizations targeted for certain
// common patterns, rather than using the slower, but more general logic.
// All tests should pass regardless of whether this is true or not.
const optimizeCommon = true

var (
	// Most natural Go type that correspond with each JSON type.
	anyType          = reflect.TypeOf((*any)(nil)).Elem()            // JSON value
	boolType         = reflect.TypeOf((*bool)(nil)).Elem()           // JSON bool
	stringType       = reflect.TypeOf((*string)(nil)).Elem()         // JSON string
	float64Type      = reflect.TypeOf((*float64)(nil)).Elem()        // JSON number
	mapStringAnyType = reflect.TypeOf((*map[string]any)(nil)).Elem() // JSON object
	sliceAnyType     = reflect.TypeOf((*[]any)(nil)).Elem()          // JSON array

	bytesType       = reflect.TypeOf((*[]byte)(nil)).Elem()
	emptyStructType = reflect.TypeOf((*struct{})(nil)).Elem()
)

const startDetectingCyclesAfter = 1000

type seenPointers = map[any]struct{}

type typedPointer struct {
	typ reflect.Type
	ptr any // always stores unsafe.Pointer, but avoids depending on unsafe
	len int // remember slice length to avoid false positives
}

// visitPointer visits pointer p of type t, reporting an error if seen before.
// If successfully visited, then the caller must eventually call leave.
func visitPointer(m *seenPointers, v reflect.Value) error {
	p := typedPointer{v.Type(), v.UnsafePointer(), sliceLen(v)}
	if _, ok := (*m)[p]; ok {
		return &SemanticError{action: "marshal", GoType: p.typ, Err: errors.New("encountered a cycle")}
	}
	if *m == nil {
		*m = make(seenPointers)
	}
	(*m)[p] = struct{}{}
	return nil
}
func leavePointer(m *seenPointers, v reflect.Value) {
	p := typedPointer{v.Type(), v.UnsafePointer(), sliceLen(v)}
	delete(*m, p)
}

func sliceLen(v reflect.Value) int {
	if v.Kind() == reflect.Slice {
		return v.Len()
	}
	return 0
}

func len64[Bytes ~[]byte | ~string](in Bytes) int64 {
	return int64(len(in))
}

func makeDefaultArshaler(t reflect.Type) *arshaler {
	switch t.Kind() {
	case reflect.Bool:
		return makeBoolArshaler(t)
	case reflect.String:
		return makeStringArshaler(t)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return makeIntArshaler(t)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return makeUintArshaler(t)
	case reflect.Float32, reflect.Float64:
		return makeFloatArshaler(t)
	case reflect.Map:
		return makeMapArshaler(t)
	case reflect.Struct:
		return makeStructArshaler(t)
	case reflect.Slice:
		fncs := makeSliceArshaler(t)
		if t.AssignableTo(bytesType) {
			return makeBytesArshaler(t, fncs)
		}
		return fncs
	case reflect.Array:
		fncs := makeArrayArshaler(t)
		if reflect.SliceOf(t.Elem()).AssignableTo(bytesType) {
			return makeBytesArshaler(t, fncs)
		}
		return fncs
	case reflect.Pointer:
		return makePointerArshaler(t)
	case reflect.Interface:
		return makeInterfaceArshaler(t)
	default:
		return makeInvalidArshaler(t)
	}
}

func makeBoolArshaler(t reflect.Type) *arshaler {
	var fncs arshaler
	fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
		xe := export.Encoder(enc)
		if mo.Format != "" && mo.FormatDepth == xe.Tokens.Depth() {
			return newInvalidFormatError("marshal", t, mo.Format)
		}

		// Optimize for marshaling without preceding whitespace.
		if optimizeCommon && !xe.Flags.Get(jsonflags.Expand) && !xe.Tokens.Last.NeedObjectName() {
			xe.Buf = strconv.AppendBool(xe.Tokens.MayAppendDelim(xe.Buf, 't'), va.Bool())
			xe.Tokens.Last.Increment()
			if xe.NeedFlush() {
				return xe.Flush()
			}
			return nil
		}

		return enc.WriteToken(jsontext.Bool(va.Bool()))
	}
	fncs.unmarshal = func(dec *jsontext.Decoder, va addressableValue, uo *jsonopts.Struct) error {
		xd := export.Decoder(dec)
		if uo.Format != "" && uo.FormatDepth == xd.Tokens.Depth() {
			return newInvalidFormatError("unmarshal", t, uo.Format)
		}
		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		k := tok.Kind()
		switch k {
		case 'n':
			va.SetBool(false)
			return nil
		case 't', 'f':
			va.SetBool(tok.Bool())
			return nil
		}
		return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t}
	}
	return &fncs
}

func makeStringArshaler(t reflect.Type) *arshaler {
	var fncs arshaler
	fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
		xe := export.Encoder(enc)
		if mo.Format != "" && mo.FormatDepth == xe.Tokens.Depth() {
			return newInvalidFormatError("marshal", t, mo.Format)
		}

		// Optimize for marshaling without preceding whitespace or string escaping.
		s := va.String()
		if optimizeCommon && !xe.Flags.Get(jsonflags.Expand) && !xe.Tokens.Last.NeedObjectName() && !jsonwire.NeedEscape(s) {
			b := xe.Buf
			b = xe.Tokens.MayAppendDelim(b, '"')
			b = append(b, '"')
			b = append(b, s...)
			b = append(b, '"')
			xe.Buf = b
			xe.Tokens.Last.Increment()
			if xe.NeedFlush() {
				return xe.Flush()
			}
			return nil
		}

		return enc.WriteToken(jsontext.String(s))
	}
	fncs.unmarshal = func(dec *jsontext.Decoder, va addressableValue, uo *jsonopts.Struct) error {
		xd := export.Decoder(dec)
		if uo.Format != "" && uo.FormatDepth == xd.Tokens.Depth() {
			return newInvalidFormatError("unmarshal", t, uo.Format)
		}
		var flags jsonwire.ValueFlags
		val, err := xd.ReadValue(&flags)
		if err != nil {
			return err
		}
		k := val.Kind()
		switch k {
		case 'n':
			va.SetString("")
			return nil
		case '"':
			val = jsonwire.UnquoteMayCopy(val, flags.IsVerbatim())
			if xd.StringCache == nil {
				xd.StringCache = new(stringCache)
			}
			str := makeString(xd.StringCache, val)
			va.SetString(str)
			return nil
		}
		return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t}
	}
	return &fncs
}

var (
	encodeBase16        = func(dst, src []byte) { hex.Encode(dst, src) }
	encodeBase32        = base32.StdEncoding.Encode
	encodeBase32Hex     = base32.HexEncoding.Encode
	encodeBase64        = base64.StdEncoding.Encode
	encodeBase64URL     = base64.URLEncoding.Encode
	encodedLenBase16    = hex.EncodedLen
	encodedLenBase32    = base32.StdEncoding.EncodedLen
	encodedLenBase32Hex = base32.HexEncoding.EncodedLen
	encodedLenBase64    = base64.StdEncoding.EncodedLen
	encodedLenBase64URL = base64.URLEncoding.EncodedLen
	decodeBase16        = hex.Decode
	decodeBase32        = base32.StdEncoding.Decode
	decodeBase32Hex     = base32.HexEncoding.Decode
	decodeBase64        = base64.StdEncoding.Decode
	decodeBase64URL     = base64.URLEncoding.Decode
	decodedLenBase16    = hex.DecodedLen
	decodedLenBase32    = base32.StdEncoding.WithPadding(base32.NoPadding).DecodedLen
	decodedLenBase32Hex = base32.HexEncoding.WithPadding(base32.NoPadding).DecodedLen
	decodedLenBase64    = base64.StdEncoding.WithPadding(base64.NoPadding).DecodedLen
	decodedLenBase64URL = base64.URLEncoding.WithPadding(base64.NoPadding).DecodedLen
)

func makeBytesArshaler(t reflect.Type, fncs *arshaler) *arshaler {
	// NOTE: This handles both []byte and [N]byte.
	marshalArray := fncs.marshal
	fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
		xe := export.Encoder(enc)
		encode, encodedLen := encodeBase64, encodedLenBase64
		if mo.Format != "" && mo.FormatDepth == xe.Tokens.Depth() {
			switch mo.Format {
			case "base64":
				encode, encodedLen = encodeBase64, encodedLenBase64
			case "base64url":
				encode, encodedLen = encodeBase64URL, encodedLenBase64URL
			case "base32":
				encode, encodedLen = encodeBase32, encodedLenBase32
			case "base32hex":
				encode, encodedLen = encodeBase32Hex, encodedLenBase32Hex
			case "base16", "hex":
				encode, encodedLen = encodeBase16, encodedLenBase16
			case "array":
				mo.Format = ""
				return marshalArray(enc, va, mo)
			default:
				return newInvalidFormatError("marshal", t, mo.Format)
			}
		} else if mo.Flags.Get(jsonflags.FormatByteArrayAsArray) && va.Kind() == reflect.Array {
			return marshalArray(enc, va, mo)
		}
		if mo.Flags.Get(jsonflags.FormatNilSliceAsNull) && va.Kind() == reflect.Slice && va.IsNil() {
			// TODO: Provide a "emitempty" format override?
			return enc.WriteToken(jsontext.Null)
		}
		val := enc.UnusedBuffer()
		b := va.Bytes()
		n := len(`"`) + encodedLen(len(b)) + len(`"`)
		if cap(val) < n {
			val = make([]byte, n)
		} else {
			val = val[:n]
		}
		val[0] = '"'
		encode(val[len(`"`):len(val)-len(`"`)], b)
		val[len(val)-1] = '"'
		return enc.WriteValue(val)
	}
	unmarshalArray := fncs.unmarshal
	fncs.unmarshal = func(dec *jsontext.Decoder, va addressableValue, uo *jsonopts.Struct) error {
		xd := export.Decoder(dec)
		decode, decodedLen, encodedLen := decodeBase64, decodedLenBase64, encodedLenBase64
		if uo.Format != "" && uo.FormatDepth == xd.Tokens.Depth() {
			switch uo.Format {
			case "base64":
				decode, decodedLen, encodedLen = decodeBase64, decodedLenBase64, encodedLenBase64
			case "base64url":
				decode, decodedLen, encodedLen = decodeBase64URL, decodedLenBase64URL, encodedLenBase64URL
			case "base32":
				decode, decodedLen, encodedLen = decodeBase32, decodedLenBase32, encodedLenBase32
			case "base32hex":
				decode, decodedLen, encodedLen = decodeBase32Hex, decodedLenBase32Hex, encodedLenBase32Hex
			case "base16", "hex":
				decode, decodedLen, encodedLen = decodeBase16, decodedLenBase16, encodedLenBase16
			case "array":
				uo.Format = ""
				return unmarshalArray(dec, va, uo)
			default:
				return newInvalidFormatError("unmarshal", t, uo.Format)
			}
		} else if uo.Flags.Get(jsonflags.FormatByteArrayAsArray) && va.Kind() == reflect.Array {
			return unmarshalArray(dec, va, uo)
		}
		var flags jsonwire.ValueFlags
		val, err := xd.ReadValue(&flags)
		if err != nil {
			return err
		}
		k := val.Kind()
		switch k {
		case 'n':
			va.SetZero()
			return nil
		case '"':
			val = jsonwire.UnquoteMayCopy(val, flags.IsVerbatim())

			// For base64 and base32, decodedLen computes the maximum output size
			// when given the original input size. To compute the exact size,
			// adjust the input size by excluding trailing padding characters.
			// This is unnecessary for base16, but also harmless.
			n := len(val)
			for n > 0 && val[n-1] == '=' {
				n--
			}
			n = decodedLen(n)
			b := va.Bytes()
			if va.Kind() == reflect.Array {
				if n != len(b) {
					err := fmt.Errorf("decoded base64 length of %d mismatches array length of %d", n, len(b))
					return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t, Err: err}
				}
			} else {
				if b == nil || cap(b) < n {
					b = make([]byte, n)
				} else {
					b = b[:n]
				}
			}
			n2, err := decode(b, val)
			if err == nil && len(val) != encodedLen(n2) {
				// TODO(https://go.dev/issue/53845): RFC 4648, section 3.3,
				// specifies that non-alphabet characters must be rejected.
				// Unfortunately, the "base32" and "base64" packages allow
				// '\r' and '\n' characters by default.
				err = errors.New("illegal data at input byte " + strconv.Itoa(bytes.IndexAny(val, "\r\n")))
			}
			if err != nil {
				return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t, Err: err}
			}
			if va.Kind() == reflect.Slice {
				va.SetBytes(b)
			}
			return nil
		}
		return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t}
	}
	return fncs
}

func makeIntArshaler(t reflect.Type) *arshaler {
	var fncs arshaler
	bits := t.Bits()
	fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
		xe := export.Encoder(enc)
		if mo.Format != "" && mo.FormatDepth == xe.Tokens.Depth() {
			return newInvalidFormatError("marshal", t, mo.Format)
		}

		// Optimize for marshaling without preceding whitespace or string escaping.
		if optimizeCommon && !xe.Flags.Get(jsonflags.Expand) && !mo.Flags.Get(jsonflags.StringifyNumbers) && !xe.Tokens.Last.NeedObjectName() {
			xe.Buf = strconv.AppendInt(xe.Tokens.MayAppendDelim(xe.Buf, '0'), va.Int(), 10)
			xe.Tokens.Last.Increment()
			if xe.NeedFlush() {
				return xe.Flush()
			}
			return nil
		}

		k := stringOrNumberKind(mo.Flags.Get(jsonflags.StringifyNumbers))
		return xe.AppendRaw(k, true, func(b []byte) ([]byte, error) {
			return strconv.AppendInt(b, va.Int(), 10), nil
		})
	}
	fncs.unmarshal = func(dec *jsontext.Decoder, va addressableValue, uo *jsonopts.Struct) error {
		xd := export.Decoder(dec)
		if uo.Format != "" && uo.FormatDepth == xd.Tokens.Depth() {
			return newInvalidFormatError("unmarshal", t, uo.Format)
		}
		var flags jsonwire.ValueFlags
		val, err := xd.ReadValue(&flags)
		if err != nil {
			return err
		}
		k := val.Kind()
		switch k {
		case 'n':
			va.SetInt(0)
			return nil
		case '"':
			if !uo.Flags.Get(jsonflags.StringifyNumbers) {
				break
			}
			val = jsonwire.UnquoteMayCopy(val, flags.IsVerbatim())
			fallthrough
		case '0':
			var negOffset int
			neg := len(val) > 0 && val[0] == '-'
			if neg {
				negOffset = 1
			}
			n, ok := jsonwire.ParseUint(val[negOffset:])
			maxInt := uint64(1) << (bits - 1)
			overflow := (neg && n > maxInt) || (!neg && n > maxInt-1)
			if !ok {
				if n != math.MaxUint64 {
					err := fmt.Errorf("cannot parse %q as signed integer: %w", val, strconv.ErrSyntax)
					return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t, Err: err}
				}
				overflow = true
			}
			if overflow {
				err := fmt.Errorf("cannot parse %q as signed integer: %w", val, strconv.ErrRange)
				return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t, Err: err}
			}
			if neg {
				va.SetInt(int64(-n))
			} else {
				va.SetInt(int64(+n))
			}
			return nil
		}
		return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t}
	}
	return &fncs
}

func makeUintArshaler(t reflect.Type) *arshaler {
	var fncs arshaler
	bits := t.Bits()
	fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
		xe := export.Encoder(enc)
		if mo.Format != "" && mo.FormatDepth == xe.Tokens.Depth() {
			return newInvalidFormatError("marshal", t, mo.Format)
		}

		// Optimize for marshaling without preceding whitespace or string escaping.
		if optimizeCommon && !xe.Flags.Get(jsonflags.Expand) && !mo.Flags.Get(jsonflags.StringifyNumbers) && !xe.Tokens.Last.NeedObjectName() {
			xe.Buf = strconv.AppendUint(xe.Tokens.MayAppendDelim(xe.Buf, '0'), va.Uint(), 10)
			xe.Tokens.Last.Increment()
			if xe.NeedFlush() {
				return xe.Flush()
			}
			return nil
		}

		k := stringOrNumberKind(mo.Flags.Get(jsonflags.StringifyNumbers))
		return xe.AppendRaw(k, true, func(b []byte) ([]byte, error) {
			return strconv.AppendUint(b, va.Uint(), 10), nil
		})
	}
	fncs.unmarshal = func(dec *jsontext.Decoder, va addressableValue, uo *jsonopts.Struct) error {
		xd := export.Decoder(dec)
		if uo.Format != "" && uo.FormatDepth == xd.Tokens.Depth() {
			return newInvalidFormatError("unmarshal", t, uo.Format)
		}
		var flags jsonwire.ValueFlags
		val, err := xd.ReadValue(&flags)
		if err != nil {
			return err
		}
		k := val.Kind()
		switch k {
		case 'n':
			va.SetUint(0)
			return nil
		case '"':
			if !uo.Flags.Get(jsonflags.StringifyNumbers) {
				break
			}
			val = jsonwire.UnquoteMayCopy(val, flags.IsVerbatim())
			fallthrough
		case '0':
			n, ok := jsonwire.ParseUint(val)
			maxUint := uint64(1) << bits
			overflow := n > maxUint-1
			if !ok {
				if n != math.MaxUint64 {
					err := fmt.Errorf("cannot parse %q as unsigned integer: %w", val, strconv.ErrSyntax)
					return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t, Err: err}
				}
				overflow = true
			}
			if overflow {
				err := fmt.Errorf("cannot parse %q as unsigned integer: %w", val, strconv.ErrRange)
				return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t, Err: err}
			}
			va.SetUint(n)
			return nil
		}
		return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t}
	}
	return &fncs
}

func makeFloatArshaler(t reflect.Type) *arshaler {
	var fncs arshaler
	bits := t.Bits()
	fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
		xe := export.Encoder(enc)
		var allowNonFinite bool
		if mo.Format != "" && mo.FormatDepth == xe.Tokens.Depth() {
			if mo.Format == "nonfinite" {
				allowNonFinite = true
			} else {
				return newInvalidFormatError("marshal", t, mo.Format)
			}
		}

		fv := va.Float()
		if math.IsNaN(fv) || math.IsInf(fv, 0) {
			if !allowNonFinite {
				err := fmt.Errorf("invalid value: %v", fv)
				return &SemanticError{action: "marshal", GoType: t, Err: err}
			}
			return enc.WriteToken(jsontext.Float(fv))
		}

		// Optimize for marshaling without preceding whitespace or string escaping.
		if optimizeCommon && !xe.Flags.Get(jsonflags.Expand) && !mo.Flags.Get(jsonflags.StringifyNumbers) && !xe.Tokens.Last.NeedObjectName() {
			xe.Buf = jsonwire.AppendFloat(xe.Tokens.MayAppendDelim(xe.Buf, '0'), fv, bits)
			xe.Tokens.Last.Increment()
			if xe.NeedFlush() {
				return xe.Flush()
			}
			return nil
		}

		k := stringOrNumberKind(mo.Flags.Get(jsonflags.StringifyNumbers))
		return xe.AppendRaw(k, true, func(b []byte) ([]byte, error) {
			return jsonwire.AppendFloat(b, va.Float(), bits), nil
		})
	}
	fncs.unmarshal = func(dec *jsontext.Decoder, va addressableValue, uo *jsonopts.Struct) error {
		xd := export.Decoder(dec)
		var allowNonFinite bool
		if uo.Format != "" && uo.FormatDepth == xd.Tokens.Depth() {
			if uo.Format == "nonfinite" {
				allowNonFinite = true
			} else {
				return newInvalidFormatError("unmarshal", t, uo.Format)
			}
		}
		var flags jsonwire.ValueFlags
		val, err := xd.ReadValue(&flags)
		if err != nil {
			return err
		}
		k := val.Kind()
		switch k {
		case 'n':
			va.SetFloat(0)
			return nil
		case '"':
			val = jsonwire.UnquoteMayCopy(val, flags.IsVerbatim())
			if allowNonFinite {
				switch string(val) {
				case "NaN":
					va.SetFloat(math.NaN())
					return nil
				case "Infinity":
					va.SetFloat(math.Inf(+1))
					return nil
				case "-Infinity":
					va.SetFloat(math.Inf(-1))
					return nil
				}
			}
			if !uo.Flags.Get(jsonflags.StringifyNumbers) {
				break
			}
			if n, err := jsonwire.ConsumeNumber(val); n != len(val) || err != nil {
				err := fmt.Errorf("cannot parse %q as JSON number: %w", val, strconv.ErrSyntax)
				return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t, Err: err}
			}
			fallthrough
		case '0':
			fv, ok := jsonwire.ParseFloat(val, bits)
			if !ok && uo.Flags.Get(jsonflags.RejectFloatOverflow) {
				return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t, Err: strconv.ErrRange}
			}
			va.SetFloat(fv)
			return nil
		}
		return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t}
	}
	return &fncs
}

func makeMapArshaler(t reflect.Type) *arshaler {
	// NOTE: The logic below disables namespaces for tracking duplicate names
	// when handling map keys with a unique representation.

	// NOTE: Values retrieved from a map are not addressable,
	// so we shallow copy the values to make them addressable and
	// store them back into the map afterwards.

	var fncs arshaler
	var (
		once    sync.Once
		keyFncs *arshaler
		valFncs *arshaler
	)
	init := func() {
		keyFncs = lookupArshaler(t.Key())
		valFncs = lookupArshaler(t.Elem())
	}
	fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
		// Check for cycles.
		xe := export.Encoder(enc)
		if xe.Tokens.Depth() > startDetectingCyclesAfter {
			if err := visitPointer(&xe.SeenPointers, va.Value); err != nil {
				return err
			}
			defer leavePointer(&xe.SeenPointers, va.Value)
		}

		emitNull := mo.Flags.Get(jsonflags.FormatNilMapAsNull)
		if mo.Format != "" && mo.FormatDepth == xe.Tokens.Depth() {
			switch mo.Format {
			case "emitnull":
				emitNull = true
				mo.Format = ""
			case "emitempty":
				emitNull = false
				mo.Format = ""
			default:
				return newInvalidFormatError("marshal", t, mo.Format)
			}
		}

		// Handle empty maps.
		n := va.Len()
		if n == 0 {
			if emitNull && va.IsNil() {
				return enc.WriteToken(jsontext.Null)
			}
			// Optimize for marshaling an empty map without any preceding whitespace.
			if optimizeCommon && !xe.Flags.Get(jsonflags.Expand) && !xe.Tokens.Last.NeedObjectName() {
				xe.Buf = append(xe.Tokens.MayAppendDelim(xe.Buf, '{'), "{}"...)
				xe.Tokens.Last.Increment()
				if xe.NeedFlush() {
					return xe.Flush()
				}
				return nil
			}
		}

		once.Do(init)
		if err := enc.WriteToken(jsontext.ObjectStart); err != nil {
			return err
		}
		if n > 0 {
			nonDefaultKey := keyFncs.nonDefault
			marshalKey := keyFncs.marshal
			marshalVal := valFncs.marshal
			if mo.Marshalers != nil {
				var ok bool
				marshalKey, ok = mo.Marshalers.(*Marshalers).lookup(marshalKey, t.Key())
				marshalVal, _ = mo.Marshalers.(*Marshalers).lookup(marshalVal, t.Elem())
				nonDefaultKey = nonDefaultKey || ok
			}
			k := newAddressableValue(t.Key())
			v := newAddressableValue(t.Elem())

			// A Go map guarantees that each entry has a unique key.
			// As such, disable the expensive duplicate name check if we know
			// that every Go key will serialize as a unique JSON string.
			if !nonDefaultKey && mapKeyWithUniqueRepresentation(k.Kind(), xe.Flags.Get(jsonflags.AllowInvalidUTF8)) {
				xe.Tokens.Last.DisableNamespace()
			}

			switch {
			case !mo.Flags.Get(jsonflags.Deterministic) || n <= 1:
				for iter := va.Value.MapRange(); iter.Next(); {
					k.SetIterKey(iter)
					flagsOriginal := mo.Flags
					mo.Flags.Set(jsonflags.StringifyNumbers | 1) // stringify for numeric keys
					err := marshalKey(enc, k, mo)
					mo.Flags = flagsOriginal
					if err != nil {
						// TODO: If err is errMissingName, then wrap it as a
						// SemanticError since this key type cannot be serialized
						// as a JSON string.
						return err
					}
					v.SetIterValue(iter)
					if err := marshalVal(enc, v, mo); err != nil {
						return err
					}
				}
			case !nonDefaultKey && t.Key().Kind() == reflect.String:
				names := getStrings(n)
				for i, iter := 0, va.Value.MapRange(); i < n && iter.Next(); i++ {
					k.SetIterKey(iter)
					(*names)[i] = k.String()
				}
				names.Sort()
				for _, name := range *names {
					if err := enc.WriteToken(jsontext.String(name)); err != nil {
						return err
					}
					// TODO(https://go.dev/issue/57061): Use v.SetMapIndexOf.
					k.SetString(name)
					v.Set(va.MapIndex(k.Value))
					if err := marshalVal(enc, v, mo); err != nil {
						return err
					}
				}
				putStrings(names)
			default:
				type member struct {
					name string // unquoted name
					key  addressableValue
					val  addressableValue
				}
				members := make([]member, n)
				keys := reflect.MakeSlice(reflect.SliceOf(t.Key()), n, n)
				vals := reflect.MakeSlice(reflect.SliceOf(t.Elem()), n, n)
				for i, iter := 0, va.Value.MapRange(); i < n && iter.Next(); i++ {
					// Marshal the member name.
					k := addressableValue{keys.Index(i)} // indexed slice element is always addressable
					k.SetIterKey(iter)
					v := addressableValue{vals.Index(i)} // indexed slice element is always addressable
					v.SetIterValue(iter)
					flagsOriginal := mo.Flags
					mo.Flags.Set(jsonflags.StringifyNumbers | 1) // stringify for numeric keys
					err := marshalKey(enc, k, mo)
					mo.Flags = flagsOriginal
					if err != nil {
						// TODO: If err is errMissingName, then wrap it as a
						// SemanticError since this key type cannot be serialized
						// as a JSON string.
						return err
					}
					name := xe.UnwriteOnlyObjectMemberName()
					members[i] = member{name, k, v}
				}
				// TODO: If AllowDuplicateNames is enabled, then sort according
				// to reflect.Value as well if the names are equal.
				// See internal/fmtsort.
				slices.SortFunc(members, func(x, y member) int {
					return jsonwire.CompareUTF16(x.name, y.name)
				})
				for _, member := range members {
					if err := enc.WriteToken(jsontext.String(member.name)); err != nil {
						return err
					}
					if err := marshalVal(enc, member.val, mo); err != nil {
						return err
					}
				}
			}
		}
		if err := enc.WriteToken(jsontext.ObjectEnd); err != nil {
			return err
		}
		return nil
	}
	fncs.unmarshal = func(dec *jsontext.Decoder, va addressableValue, uo *jsonopts.Struct) error {
		xd := export.Decoder(dec)
		if uo.Format != "" && uo.FormatDepth == xd.Tokens.Depth() {
			switch uo.Format {
			case "emitnull", "emitempty":
				uo.Format = "" // only relevant for marshaling
			default:
				return newInvalidFormatError("unmarshal", t, uo.Format)
			}
		}
		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		k := tok.Kind()
		switch k {
		case 'n':
			va.SetZero()
			return nil
		case '{':
			once.Do(init)
			if va.IsNil() {
				va.Set(reflect.MakeMap(t))
			}

			nonDefaultKey := keyFncs.nonDefault
			unmarshalKey := keyFncs.unmarshal
			unmarshalVal := valFncs.unmarshal
			if uo.Unmarshalers != nil {
				var ok bool
				unmarshalKey, ok = uo.Unmarshalers.(*Unmarshalers).lookup(unmarshalKey, t.Key())
				unmarshalVal, _ = uo.Unmarshalers.(*Unmarshalers).lookup(unmarshalVal, t.Elem())
				nonDefaultKey = nonDefaultKey || ok
			}
			k := newAddressableValue(t.Key())
			v := newAddressableValue(t.Elem())

			// Manually check for duplicate entries by virtue of whether the
			// unmarshaled key already exists in the destination Go map.
			// Consequently, syntactically different names (e.g., "0" and "-0")
			// will be rejected as duplicates since they semantically refer
			// to the same Go value. This is an unusual interaction
			// between syntax and semantics, but is more correct.
			if !nonDefaultKey && mapKeyWithUniqueRepresentation(k.Kind(), xd.Flags.Get(jsonflags.AllowInvalidUTF8)) {
				xd.Tokens.Last.DisableNamespace()
			}

			// In the rare case where the map is not already empty,
			// then we need to manually track which keys we already saw
			// since existing presence alone is insufficient to indicate
			// whether the input had a duplicate name.
			var seen reflect.Value
			if !xd.Flags.Get(jsonflags.AllowDuplicateNames) && va.Len() > 0 {
				seen = reflect.MakeMap(reflect.MapOf(k.Type(), emptyStructType))
			}

			for dec.PeekKind() != '}' {
				k.SetZero()
				flagsOriginal := uo.Flags
				uo.Flags.Set(jsonflags.StringifyNumbers | 1) // stringify for numeric keys
				err := unmarshalKey(dec, k, uo)
				uo.Flags = flagsOriginal
				if err != nil {
					return err
				}
				if k.Kind() == reflect.Interface && !k.IsNil() && !k.Elem().Type().Comparable() {
					err := fmt.Errorf("invalid incomparable key type %v", k.Elem().Type())
					return &SemanticError{action: "unmarshal", GoType: t, Err: err}
				}

				if v2 := va.MapIndex(k.Value); v2.IsValid() {
					if !xd.Flags.Get(jsonflags.AllowDuplicateNames) && (!seen.IsValid() || seen.MapIndex(k.Value).IsValid()) {
						// TODO: Unread the object name.
						name := xd.PreviousBuffer()
						err := export.NewDuplicateNameError(name, dec.InputOffset()-len64(name))
						return err
					}
					v.Set(v2)
				} else {
					v.SetZero()
				}
				err = unmarshalVal(dec, v, uo)
				va.SetMapIndex(k.Value, v.Value)
				if seen.IsValid() {
					seen.SetMapIndex(k.Value, reflect.Zero(emptyStructType))
				}
				if err != nil {
					return err
				}
			}
			if _, err := dec.ReadToken(); err != nil {
				return err
			}
			return nil
		}
		return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t}
	}
	return &fncs
}

// mapKeyWithUniqueRepresentation reports whether all possible values of k
// marshal to a different JSON value, and whether all possible JSON values
// that can unmarshal into k unmarshal to different Go values.
// In other words, the representation must be a bijective.
func mapKeyWithUniqueRepresentation(k reflect.Kind, allowInvalidUTF8 bool) bool {
	switch k {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	case reflect.String:
		// For strings, we have to be careful since names with invalid UTF-8
		// maybe unescape to the same Go string value.
		return !allowInvalidUTF8
	default:
		// Floating-point kinds are not listed above since NaNs
		// can appear multiple times and all serialize as "NaN".
		return false
	}
}

func makeStructArshaler(t reflect.Type) *arshaler {
	// NOTE: The logic below disables namespaces for tracking duplicate names
	// and does the tracking locally with an efficient bit-set based on which
	// Go struct fields were seen.

	var fncs arshaler
	var (
		once    sync.Once
		fields  structFields
		errInit *SemanticError
	)
	init := func() {
		fields, errInit = makeStructFields(t)
	}
	fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
		xe := export.Encoder(enc)
		if mo.Format != "" && mo.FormatDepth == xe.Tokens.Depth() {
			return newInvalidFormatError("marshal", t, mo.Format)
		}
		once.Do(init)
		if errInit != nil {
			err := *errInit // shallow copy SemanticError
			err.action = "marshal"
			return &err
		}
		if err := enc.WriteToken(jsontext.ObjectStart); err != nil {
			return err
		}
		var seenIdxs uintSet
		prevIdx := -1
		xe.Tokens.Last.DisableNamespace() // we manually ensure unique names below
		for i := range fields.flattened {
			f := &fields.flattened[i]
			v := addressableValue{va.Field(f.index[0])} // addressable if struct value is addressable
			if len(f.index) > 1 {
				v = v.fieldByIndex(f.index[1:], false)
				if !v.IsValid() {
					continue // implies a nil inlined field
				}
			}

			// OmitZero skips the field if the Go value is zero,
			// which we can determine up front without calling the marshaler.
			if f.omitzero && ((f.isZero == nil && v.IsZero()) || (f.isZero != nil && f.isZero(v))) {
				continue
			}

			// Check for the legacy definition of omitempty.
			if f.omitempty && mo.Flags.Get(jsonflags.OmitEmptyWithLegacyDefinition) && isLegacyEmpty(v) {
				continue
			}

			marshal := f.fncs.marshal
			nonDefault := f.fncs.nonDefault
			if mo.Marshalers != nil {
				var ok bool
				marshal, ok = mo.Marshalers.(*Marshalers).lookup(marshal, f.typ)
				nonDefault = nonDefault || ok
			}

			// OmitEmpty skips the field if the marshaled JSON value is empty,
			// which we can know up front if there are no custom marshalers,
			// otherwise we must marshal the value and unwrite it if empty.
			if f.omitempty && !mo.Flags.Get(jsonflags.OmitEmptyWithLegacyDefinition) &&
				!nonDefault && f.isEmpty != nil && f.isEmpty(v) {
				continue // fast path for omitempty
			}

			// Write the object member name.
			//
			// The logic below is semantically equivalent to:
			//	enc.WriteToken(String(f.name))
			// but specialized and simplified because:
			//	1. The Encoder must be expecting an object name.
			//	2. The object namespace is guaranteed to be disabled.
			//	3. The object name is guaranteed to be valid and pre-escaped.
			//	4. There is no need to flush the buffer (for unwrite purposes).
			//	5. There is no possibility of an error occurring.
			if optimizeCommon {
				// Append any delimiters or optional whitespace.
				b := xe.Buf
				if xe.Tokens.Last.Length() > 0 {
					b = append(b, ',')
				}
				if xe.Flags.Get(jsonflags.Expand) {
					b = xe.AppendIndent(b, xe.Tokens.NeedIndent('"'))
				}

				// Append the token to the output and to the state machine.
				n0 := len(b) // offset before calling AppendQuote
				if !xe.Flags.Get(jsonflags.EscapeForHTML | jsonflags.EscapeForJS) {
					b = append(b, f.quotedName...)
				} else {
					b, _ = jsonwire.AppendQuote(b, f.name, &xe.Flags)
				}
				xe.Buf = b
				if !xe.Flags.Get(jsonflags.AllowDuplicateNames) {
					xe.Names.ReplaceLastQuotedOffset(n0)
				}
				xe.Tokens.Last.Increment()
			} else {
				if err := enc.WriteToken(jsontext.String(f.name)); err != nil {
					return err
				}
			}

			// Write the object member value.
			flagsOriginal := mo.Flags
			if f.string {
				mo.Flags.Set(jsonflags.StringifyNumbers | 1)
			}
			if f.format != "" {
				mo.FormatDepth = xe.Tokens.Depth()
				mo.Format = f.format
			}
			err := marshal(enc, v, mo)
			mo.Flags = flagsOriginal
			mo.Format = ""
			if err != nil {
				return err
			}

			// Try unwriting the member if empty (slow path for omitempty).
			if f.omitempty && !mo.Flags.Get(jsonflags.OmitEmptyWithLegacyDefinition) {
				var prevName *string
				if prevIdx >= 0 {
					prevName = &fields.flattened[prevIdx].name
				}
				if xe.UnwriteEmptyObjectMember(prevName) {
					continue
				}
			}

			// Remember the previous written object member.
			// The set of seen fields only needs to be updated to detect
			// duplicate names with those from the inlined fallback.
			if !xe.Flags.Get(jsonflags.AllowDuplicateNames) && fields.inlinedFallback != nil {
				seenIdxs.insert(uint(f.id))
			}
			prevIdx = f.id
		}
		if fields.inlinedFallback != nil && !(mo.Flags.Get(jsonflags.DiscardUnknownMembers) && fields.inlinedFallback.unknown) {
			var insertUnquotedName func([]byte) bool
			if !xe.Flags.Get(jsonflags.AllowDuplicateNames) {
				insertUnquotedName = func(name []byte) bool {
					// Check that the name from inlined fallback does not match
					// one of the previously marshaled names from known fields.
					if foldedFields := fields.lookupByFoldedName(name); len(foldedFields) > 0 {
						if f := fields.byActualName[string(name)]; f != nil {
							return seenIdxs.insert(uint(f.id))
						}
						for _, f := range foldedFields {
							if f.matchFoldedName(name, &mo.Flags) {
								return seenIdxs.insert(uint(f.id))
							}
						}
					}

					// Check that the name does not match any other name
					// previously marshaled from the inlined fallback.
					return xe.Namespaces.Last().InsertUnquoted(name)
				}
			}
			if err := marshalInlinedFallbackAll(enc, va, mo, fields.inlinedFallback, insertUnquotedName); err != nil {
				return err
			}
		}
		if err := enc.WriteToken(jsontext.ObjectEnd); err != nil {
			return err
		}
		return nil
	}
	fncs.unmarshal = func(dec *jsontext.Decoder, va addressableValue, uo *jsonopts.Struct) error {
		xd := export.Decoder(dec)
		if uo.Format != "" && uo.FormatDepth == xd.Tokens.Depth() {
			return newInvalidFormatError("unmarshal", t, uo.Format)
		}
		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		k := tok.Kind()
		switch k {
		case 'n':
			va.SetZero()
			return nil
		case '{':
			once.Do(init)
			if errInit != nil {
				err := *errInit // shallow copy SemanticError
				err.action = "unmarshal"
				return &err
			}
			var seenIdxs uintSet
			xd.Tokens.Last.DisableNamespace()
			for dec.PeekKind() != '}' {
				// Process the object member name.
				var flags jsonwire.ValueFlags
				val, err := xd.ReadValue(&flags)
				if err != nil {
					return err
				}
				name := jsonwire.UnquoteMayCopy(val, flags.IsVerbatim())
				f := fields.byActualName[string(name)]
				if f == nil {
					for _, f2 := range fields.lookupByFoldedName(name) {
						if f2.matchFoldedName(name, &uo.Flags) {
							f = f2
							break
						}
					}
					if f == nil {
						if uo.Flags.Get(jsonflags.RejectUnknownMembers) && (fields.inlinedFallback == nil || fields.inlinedFallback.unknown) {
							return &SemanticError{action: "unmarshal", GoType: t, Err: fmt.Errorf("unknown name %s", val)}
						}
						if !xd.Flags.Get(jsonflags.AllowDuplicateNames) && !xd.Namespaces.Last().InsertUnquoted(name) {
							// TODO: Unread the object name.
							err := export.NewDuplicateNameError(val, dec.InputOffset()-len64(val))
							return err
						}

						if fields.inlinedFallback == nil {
							// Skip unknown value since we have no place to store it.
							if err := dec.SkipValue(); err != nil {
								return err
							}
						} else {
							// Marshal into value capable of storing arbitrary object members.
							if err := unmarshalInlinedFallbackNext(dec, va, uo, fields.inlinedFallback, val, name); err != nil {
								return err
							}
						}
						continue
					}
				}
				if !xd.Flags.Get(jsonflags.AllowDuplicateNames) && !seenIdxs.insert(uint(f.id)) {
					// TODO: Unread the object name.
					err := export.NewDuplicateNameError(val, dec.InputOffset()-len64(val))
					return err
				}

				// Process the object member value.
				unmarshal := f.fncs.unmarshal
				if uo.Unmarshalers != nil {
					unmarshal, _ = uo.Unmarshalers.(*Unmarshalers).lookup(unmarshal, f.typ)
				}
				flagsOriginal := uo.Flags
				if f.string {
					uo.Flags.Set(jsonflags.StringifyNumbers | 1)
				}
				if f.format != "" {
					uo.FormatDepth = xd.Tokens.Depth()
					uo.Format = f.format
				}
				v := addressableValue{va.Field(f.index[0])} // addressable if struct value is addressable
				if len(f.index) > 1 {
					v = v.fieldByIndex(f.index[1:], true)
				}
				err = unmarshal(dec, v, uo)
				uo.Flags = flagsOriginal
				uo.Format = ""
				if err != nil {
					return err
				}
			}
			if _, err := dec.ReadToken(); err != nil {
				return err
			}
			return nil
		}
		return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t}
	}
	return &fncs
}

func (va addressableValue) fieldByIndex(index []int, mayAlloc bool) addressableValue {
	for _, i := range index {
		va = va.indirect(mayAlloc)
		if !va.IsValid() {
			return va
		}
		va = addressableValue{va.Field(i)} // addressable if struct value is addressable
	}
	return va
}

func (va addressableValue) indirect(mayAlloc bool) addressableValue {
	if va.Kind() == reflect.Pointer {
		if va.IsNil() {
			if !mayAlloc {
				return addressableValue{}
			}
			va.Set(reflect.New(va.Type().Elem()))
		}
		va = addressableValue{va.Elem()} // dereferenced pointer is always addressable
	}
	return va
}

// isLegacyEmpty reports whether a value is empty according to the v1 definition.
func isLegacyEmpty(v addressableValue) bool {
	// Equivalent to encoding/json.isEmptyValue@v1.21.0.
	switch v.Kind() {
	case reflect.Bool:
		return v.Bool() == false
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.String, reflect.Map, reflect.Slice, reflect.Array:
		return v.Len() == 0
	case reflect.Pointer, reflect.Interface:
		return v.IsNil()
	}
	return false
}

func makeSliceArshaler(t reflect.Type) *arshaler {
	var fncs arshaler
	var (
		once    sync.Once
		valFncs *arshaler
	)
	init := func() {
		valFncs = lookupArshaler(t.Elem())
	}
	fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
		// Check for cycles.
		xe := export.Encoder(enc)
		if xe.Tokens.Depth() > startDetectingCyclesAfter {
			if err := visitPointer(&xe.SeenPointers, va.Value); err != nil {
				return err
			}
			defer leavePointer(&xe.SeenPointers, va.Value)
		}

		emitNull := mo.Flags.Get(jsonflags.FormatNilSliceAsNull)
		if mo.Format != "" && mo.FormatDepth == xe.Tokens.Depth() {
			switch mo.Format {
			case "emitnull":
				emitNull = true
				mo.Format = ""
			case "emitempty":
				emitNull = false
				mo.Format = ""
			default:
				return newInvalidFormatError("marshal", t, mo.Format)
			}
		}

		// Handle empty slices.
		n := va.Len()
		if n == 0 {
			if emitNull && va.IsNil() {
				return enc.WriteToken(jsontext.Null)
			}
			// Optimize for marshaling an empty slice without any preceding whitespace.
			if optimizeCommon && !xe.Flags.Get(jsonflags.Expand) && !xe.Tokens.Last.NeedObjectName() {
				xe.Buf = append(xe.Tokens.MayAppendDelim(xe.Buf, '['), "[]"...)
				xe.Tokens.Last.Increment()
				if xe.NeedFlush() {
					return xe.Flush()
				}
				return nil
			}
		}

		once.Do(init)
		if err := enc.WriteToken(jsontext.ArrayStart); err != nil {
			return err
		}
		marshal := valFncs.marshal
		if mo.Marshalers != nil {
			marshal, _ = mo.Marshalers.(*Marshalers).lookup(marshal, t.Elem())
		}
		for i := 0; i < n; i++ {
			v := addressableValue{va.Index(i)} // indexed slice element is always addressable
			if err := marshal(enc, v, mo); err != nil {
				return err
			}
		}
		if err := enc.WriteToken(jsontext.ArrayEnd); err != nil {
			return err
		}
		return nil
	}
	emptySlice := reflect.MakeSlice(t, 0, 0)
	fncs.unmarshal = func(dec *jsontext.Decoder, va addressableValue, uo *jsonopts.Struct) error {
		xd := export.Decoder(dec)
		if uo.Format != "" && uo.FormatDepth == xd.Tokens.Depth() {
			switch uo.Format {
			case "emitnull", "emitempty":
				uo.Format = "" // only relevant for marshaling
			default:
				return newInvalidFormatError("unmarshal", t, uo.Format)
			}
		}

		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		k := tok.Kind()
		switch k {
		case 'n':
			va.SetZero()
			return nil
		case '[':
			once.Do(init)
			unmarshal := valFncs.unmarshal
			if uo.Unmarshalers != nil {
				unmarshal, _ = uo.Unmarshalers.(*Unmarshalers).lookup(unmarshal, t.Elem())
			}
			mustZero := true // we do not know the cleanliness of unused capacity
			cap := va.Cap()
			if cap > 0 {
				va.SetLen(cap)
			}
			var i int
			for dec.PeekKind() != ']' {
				if i == cap {
					va.Value.Grow(1)
					cap = va.Cap()
					va.SetLen(cap)
					mustZero = false // reflect.Value.Grow ensures new capacity is zero-initialized
				}
				v := addressableValue{va.Index(i)} // indexed slice element is always addressable
				i++
				if mustZero {
					v.SetZero()
				}
				if err := unmarshal(dec, v, uo); err != nil {
					va.SetLen(i)
					return err
				}
			}
			if i == 0 {
				va.Set(emptySlice)
			} else {
				va.SetLen(i)
			}
			if _, err := dec.ReadToken(); err != nil {
				return err
			}
			return nil
		}
		return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t}
	}
	return &fncs
}

func makeArrayArshaler(t reflect.Type) *arshaler {
	var fncs arshaler
	var (
		once    sync.Once
		valFncs *arshaler
	)
	init := func() {
		valFncs = lookupArshaler(t.Elem())
	}
	n := t.Len()
	fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
		xe := export.Encoder(enc)
		if mo.Format != "" && mo.FormatDepth == xe.Tokens.Depth() {
			return newInvalidFormatError("marshal", t, mo.Format)
		}
		once.Do(init)
		if err := enc.WriteToken(jsontext.ArrayStart); err != nil {
			return err
		}
		marshal := valFncs.marshal
		if mo.Marshalers != nil {
			marshal, _ = mo.Marshalers.(*Marshalers).lookup(marshal, t.Elem())
		}
		for i := 0; i < n; i++ {
			v := addressableValue{va.Index(i)} // indexed array element is addressable if array is addressable
			if err := marshal(enc, v, mo); err != nil {
				return err
			}
		}
		if err := enc.WriteToken(jsontext.ArrayEnd); err != nil {
			return err
		}
		return nil
	}
	fncs.unmarshal = func(dec *jsontext.Decoder, va addressableValue, uo *jsonopts.Struct) error {
		xd := export.Decoder(dec)
		if uo.Format != "" && uo.FormatDepth == xd.Tokens.Depth() {
			return newInvalidFormatError("unmarshal", t, uo.Format)
		}
		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		k := tok.Kind()
		switch k {
		case 'n':
			va.SetZero()
			return nil
		case '[':
			once.Do(init)
			unmarshal := valFncs.unmarshal
			if uo.Unmarshalers != nil {
				unmarshal, _ = uo.Unmarshalers.(*Unmarshalers).lookup(unmarshal, t.Elem())
			}
			var i int
			for dec.PeekKind() != ']' {
				if i >= n {
					if uo.Flags.Get(jsonflags.UnmarshalArrayFromAnyLength) {
						if err := dec.SkipValue(); err != nil {
							return err
						}
						continue
					}
					err := errors.New("too many array elements")
					return &SemanticError{action: "unmarshal", GoType: t, Err: err}
				}
				v := addressableValue{va.Index(i)} // indexed array element is addressable if array is addressable
				v.SetZero()
				if err := unmarshal(dec, v, uo); err != nil {
					return err
				}
				i++
			}
			if _, err := dec.ReadToken(); err != nil {
				return err
			}
			if i < n {
				if uo.Flags.Get(jsonflags.UnmarshalArrayFromAnyLength) {
					for ; i < n; i++ {
						va.Index(i).SetZero()
					}
					return nil
				}
				err := errors.New("too few array elements")
				return &SemanticError{action: "unmarshal", GoType: t, Err: err}
			}
			return nil
		}
		return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t}
	}
	return &fncs
}

func makePointerArshaler(t reflect.Type) *arshaler {
	var fncs arshaler
	var (
		once    sync.Once
		valFncs *arshaler
	)
	init := func() {
		valFncs = lookupArshaler(t.Elem())
	}
	fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
		// Check for cycles.
		xe := export.Encoder(enc)
		if xe.Tokens.Depth() > startDetectingCyclesAfter {
			if err := visitPointer(&xe.SeenPointers, va.Value); err != nil {
				return err
			}
			defer leavePointer(&xe.SeenPointers, va.Value)
		}

		// NOTE: Struct.Format is forwarded to underlying marshal.
		if va.IsNil() {
			return enc.WriteToken(jsontext.Null)
		}
		once.Do(init)
		marshal := valFncs.marshal
		if mo.Marshalers != nil {
			marshal, _ = mo.Marshalers.(*Marshalers).lookup(marshal, t.Elem())
		}
		v := addressableValue{va.Elem()} // dereferenced pointer is always addressable
		return marshal(enc, v, mo)
	}
	fncs.unmarshal = func(dec *jsontext.Decoder, va addressableValue, uo *jsonopts.Struct) error {
		// NOTE: Struct.Format is forwarded to underlying unmarshal.
		if dec.PeekKind() == 'n' {
			if _, err := dec.ReadToken(); err != nil {
				return err
			}
			va.SetZero()
			return nil
		}
		once.Do(init)
		unmarshal := valFncs.unmarshal
		if uo.Unmarshalers != nil {
			unmarshal, _ = uo.Unmarshalers.(*Unmarshalers).lookup(unmarshal, t.Elem())
		}
		if va.IsNil() {
			va.Set(reflect.New(t.Elem()))
		}
		v := addressableValue{va.Elem()} // dereferenced pointer is always addressable
		return unmarshal(dec, v, uo)
	}
	return &fncs
}

func makeInterfaceArshaler(t reflect.Type) *arshaler {
	// NOTE: Values retrieved from an interface are not addressable,
	// so we shallow copy the values to make them addressable and
	// store them back into the interface afterwards.

	var fncs arshaler
	fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
		xe := export.Encoder(enc)
		if mo.Format != "" && mo.FormatDepth == xe.Tokens.Depth() {
			return newInvalidFormatError("marshal", t, mo.Format)
		}
		if va.IsNil() {
			return enc.WriteToken(jsontext.Null)
		}
		v := newAddressableValue(va.Elem().Type())
		v.Set(va.Elem())
		marshal := lookupArshaler(v.Type()).marshal
		if mo.Marshalers != nil {
			marshal, _ = mo.Marshalers.(*Marshalers).lookup(marshal, v.Type())
		}
		// Optimize for the any type if there are no special options.
		if optimizeCommon &&
			t == anyType && !mo.Flags.Get(jsonflags.StringifyNumbers) && mo.Format == "" &&
			(mo.Marshalers == nil || !mo.Marshalers.(*Marshalers).fromAny) {
			return marshalValueAny(enc, va.Elem().Interface(), mo)
		}
		return marshal(enc, v, mo)
	}
	fncs.unmarshal = func(dec *jsontext.Decoder, va addressableValue, uo *jsonopts.Struct) error {
		xd := export.Decoder(dec)
		if uo.Format != "" && uo.FormatDepth == xd.Tokens.Depth() {
			return newInvalidFormatError("unmarshal", t, uo.Format)
		}
		if dec.PeekKind() == 'n' {
			if _, err := dec.ReadToken(); err != nil {
				return err
			}
			va.SetZero()
			return nil
		}
		var v addressableValue
		if va.IsNil() {
			// Optimize for the any type if there are no special options.
			// We do not care about stringified numbers since JSON strings
			// are always unmarshaled into an any value as Go strings.
			// Duplicate name check must be enforced since unmarshalValueAny
			// does not implement merge semantics.
			if optimizeCommon &&
				t == anyType && !xd.Flags.Get(jsonflags.AllowDuplicateNames) && uo.Format == "" &&
				(uo.Unmarshalers == nil || !uo.Unmarshalers.(*Unmarshalers).fromAny) {
				v, err := unmarshalValueAny(dec, uo)
				// We must check for nil interface values up front.
				// See https://go.dev/issue/52310.
				if v != nil {
					va.Set(reflect.ValueOf(v))
				}
				return err
			}

			k := dec.PeekKind()
			if !isAnyType(t) {
				err := errors.New("cannot derive concrete type for non-empty interface")
				return &SemanticError{action: "unmarshal", JSONKind: k, GoType: t, Err: err}
			}
			switch k {
			case 'f', 't':
				v = newAddressableValue(boolType)
			case '"':
				v = newAddressableValue(stringType)
			case '0':
				v = newAddressableValue(float64Type)
			case '{':
				v = newAddressableValue(mapStringAnyType)
			case '[':
				v = newAddressableValue(sliceAnyType)
			default:
				// If k is invalid (e.g., due to an I/O or syntax error), then
				// that will be cached by PeekKind and returned by ReadValue.
				// If k is '}' or ']', then ReadValue must error since
				// those are invalid kinds at the start of a JSON value.
				_, err := dec.ReadValue()
				return err
			}
		} else {
			// Shallow copy the existing value to keep it addressable.
			// Any mutations at the top-level of the value will be observable
			// since we always store this value back into the interface value.
			v = newAddressableValue(va.Elem().Type())
			v.Set(va.Elem())
		}
		unmarshal := lookupArshaler(v.Type()).unmarshal
		if uo.Unmarshalers != nil {
			unmarshal, _ = uo.Unmarshalers.(*Unmarshalers).lookup(unmarshal, v.Type())
		}
		err := unmarshal(dec, v, uo)
		va.Set(v.Value)
		return err
	}
	return &fncs
}

// isAnyType reports wether t is equivalent to the any interface type.
func isAnyType(t reflect.Type) bool {
	// This is forward compatible if the Go language permits type sets within
	// ordinary interfaces where an interface with zero methods does not
	// necessarily mean it can hold every possible Go type.
	// See https://go.dev/issue/45346.
	return t == anyType || anyType.Implements(t)
}

func makeInvalidArshaler(t reflect.Type) *arshaler {
	var fncs arshaler
	fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
		return &SemanticError{action: "marshal", GoType: t}
	}
	fncs.unmarshal = func(dec *jsontext.Decoder, va addressableValue, uo *jsonopts.Struct) error {
		return &SemanticError{action: "unmarshal", GoType: t}
	}
	return &fncs
}

func newInvalidFormatError(action string, t reflect.Type, format string) error {
	err := fmt.Errorf("invalid format flag: %q", format)
	return &SemanticError{action: action, GoType: t, Err: err}
}

func stringOrNumberKind(isString bool) jsontext.Kind {
	if isString {
		return '"'
	} else {
		return '0'
	}
}

type uintSet64 uint64

func (s uintSet64) has(i uint) bool { return s&(1<<i) > 0 }
func (s *uintSet64) set(i uint)     { *s |= 1 << i }

// uintSet is a set of unsigned integers.
// It is optimized for most integers being close to zero.
type uintSet struct {
	lo uintSet64
	hi []uintSet64
}

// has reports whether i is in the set.
func (s *uintSet) has(i uint) bool {
	if i < 64 {
		return s.lo.has(i)
	} else {
		i -= 64
		iHi, iLo := int(i/64), i%64
		return iHi < len(s.hi) && s.hi[iHi].has(iLo)
	}
}

// insert inserts i into the set and reports whether it was the first insertion.
func (s *uintSet) insert(i uint) bool {
	// TODO: Make this inlinable at least for the lower 64-bit case.
	if i < 64 {
		has := s.lo.has(i)
		s.lo.set(i)
		return !has
	} else {
		i -= 64
		iHi, iLo := int(i/64), i%64
		if iHi >= len(s.hi) {
			s.hi = append(s.hi, make([]uintSet64, iHi+1-len(s.hi))...)
			s.hi = s.hi[:cap(s.hi)]
		}
		has := s.hi[iHi].has(iLo)
		s.hi[iHi].set(iLo)
		return !has
	}
}
