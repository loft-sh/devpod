// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"reflect"
	"strconv"

	"github.com/go-json-experiment/json/internal/jsonflags"
	"github.com/go-json-experiment/json/internal/jsonopts"
	"github.com/go-json-experiment/json/internal/jsonwire"
	"github.com/go-json-experiment/json/jsontext"
)

// This file contains an optimized marshal and unmarshal implementation
// for the any type. This type is often used when the Go program has
// no knowledge of the JSON schema. This is a common enough occurrence
// to justify the complexity of adding logic for this.

func marshalValueAny(enc *jsontext.Encoder, val any, mo *jsonopts.Struct) error {
	switch val := val.(type) {
	case nil:
		return enc.WriteToken(jsontext.Null)
	case bool:
		return enc.WriteToken(jsontext.Bool(val))
	case string:
		return enc.WriteToken(jsontext.String(val))
	case float64:
		return enc.WriteToken(jsontext.Float(val))
	case map[string]any:
		return marshalObjectAny(enc, val, mo)
	case []any:
		return marshalArrayAny(enc, val, mo)
	default:
		v := newAddressableValue(reflect.TypeOf(val))
		v.Set(reflect.ValueOf(val))
		marshal := lookupArshaler(v.Type()).marshal
		if mo.Marshalers != nil {
			marshal, _ = mo.Marshalers.(*Marshalers).lookup(marshal, v.Type())
		}
		return marshal(enc, v, mo)
	}
}

func unmarshalValueAny(dec *jsontext.Decoder, uo *jsonopts.Struct) (any, error) {
	switch k := dec.PeekKind(); k {
	case '{':
		return unmarshalObjectAny(dec, uo)
	case '[':
		return unmarshalArrayAny(dec, uo)
	default:
		xd := export.Decoder(dec)
		var flags jsonwire.ValueFlags
		val, err := xd.ReadValue(&flags)
		if err != nil {
			return nil, err
		}
		switch val.Kind() {
		case 'n':
			return nil, nil
		case 'f':
			return false, nil
		case 't':
			return true, nil
		case '"':
			val = jsonwire.UnquoteMayCopy(val, flags.IsVerbatim())
			if xd.StringCache == nil {
				xd.StringCache = new(stringCache)
			}
			return makeString(xd.StringCache, val), nil
		case '0':
			fv, ok := jsonwire.ParseFloat(val, 64)
			if !ok && uo.Flags.Get(jsonflags.RejectFloatOverflow) {
				return nil, &SemanticError{action: "unmarshal", JSONKind: k, GoType: float64Type, Err: strconv.ErrRange}
			}
			return fv, nil
		default:
			panic("BUG: invalid kind: " + k.String())
		}
	}
}

func marshalObjectAny(enc *jsontext.Encoder, obj map[string]any, mo *jsonopts.Struct) error {
	// Check for cycles.
	xe := export.Encoder(enc)
	if xe.Tokens.Depth() > startDetectingCyclesAfter {
		v := reflect.ValueOf(obj)
		if err := visitPointer(&xe.SeenPointers, v); err != nil {
			return err
		}
		defer leavePointer(&xe.SeenPointers, v)
	}

	// Handle empty maps.
	if len(obj) == 0 {
		if mo.Flags.Get(jsonflags.FormatNilMapAsNull) && obj == nil {
			return enc.WriteToken(jsontext.Null)
		}
		// Optimize for marshaling an empty map without any preceding whitespace.
		if !xe.Flags.Get(jsonflags.Expand) && !xe.Tokens.Last.NeedObjectName() {
			xe.Buf = append(xe.Tokens.MayAppendDelim(xe.Buf, '{'), "{}"...)
			xe.Tokens.Last.Increment()
			if xe.NeedFlush() {
				return xe.Flush()
			}
			return nil
		}
	}

	if err := enc.WriteToken(jsontext.ObjectStart); err != nil {
		return err
	}
	// A Go map guarantees that each entry has a unique key
	// The only possibility of duplicates is due to invalid UTF-8.
	if !xe.Flags.Get(jsonflags.AllowInvalidUTF8) {
		xe.Tokens.Last.DisableNamespace()
	}
	if !mo.Flags.Get(jsonflags.Deterministic) || len(obj) <= 1 {
		for name, val := range obj {
			if err := enc.WriteToken(jsontext.String(name)); err != nil {
				return err
			}
			if err := marshalValueAny(enc, val, mo); err != nil {
				return err
			}
		}
	} else {
		names := getStrings(len(obj))
		var i int
		for name := range obj {
			(*names)[i] = name
			i++
		}
		names.Sort()
		for _, name := range *names {
			if err := enc.WriteToken(jsontext.String(name)); err != nil {
				return err
			}
			if err := marshalValueAny(enc, obj[name], mo); err != nil {
				return err
			}
		}
		putStrings(names)
	}
	if err := enc.WriteToken(jsontext.ObjectEnd); err != nil {
		return err
	}
	return nil
}

func unmarshalObjectAny(dec *jsontext.Decoder, uo *jsonopts.Struct) (map[string]any, error) {
	tok, err := dec.ReadToken()
	if err != nil {
		return nil, err
	}
	k := tok.Kind()
	switch k {
	case 'n':
		return nil, nil
	case '{':
		xd := export.Decoder(dec)
		obj := make(map[string]any)
		// A Go map guarantees that each entry has a unique key
		// The only possibility of duplicates is due to invalid UTF-8.
		if !xd.Flags.Get(jsonflags.AllowInvalidUTF8) {
			xd.Tokens.Last.DisableNamespace()
		}
		for dec.PeekKind() != '}' {
			tok, err := dec.ReadToken()
			if err != nil {
				return obj, err
			}
			name := tok.String()

			// Manually check for duplicate names.
			if _, ok := obj[name]; ok {
				name := xd.PreviousBuffer()
				err := export.NewDuplicateNameError(name, dec.InputOffset()-len64(name))
				return obj, err
			}

			val, err := unmarshalValueAny(dec, uo)
			obj[name] = val
			if err != nil {
				return obj, err
			}
		}
		if _, err := dec.ReadToken(); err != nil {
			return obj, err
		}
		return obj, nil
	}
	return nil, &SemanticError{action: "unmarshal", JSONKind: k, GoType: mapStringAnyType}
}

func marshalArrayAny(enc *jsontext.Encoder, arr []any, mo *jsonopts.Struct) error {
	// Check for cycles.
	xe := export.Encoder(enc)
	if xe.Tokens.Depth() > startDetectingCyclesAfter {
		v := reflect.ValueOf(arr)
		if err := visitPointer(&xe.SeenPointers, v); err != nil {
			return err
		}
		defer leavePointer(&xe.SeenPointers, v)
	}

	// Handle empty slices.
	if len(arr) == 0 {
		if mo.Flags.Get(jsonflags.FormatNilSliceAsNull) && arr == nil {
			return enc.WriteToken(jsontext.Null)
		}
		// Optimize for marshaling an empty slice without any preceding whitespace.
		if !xe.Flags.Get(jsonflags.Expand) && !xe.Tokens.Last.NeedObjectName() {
			xe.Buf = append(xe.Tokens.MayAppendDelim(xe.Buf, '['), "[]"...)
			xe.Tokens.Last.Increment()
			if xe.NeedFlush() {
				return xe.Flush()
			}
			return nil
		}
	}

	if err := enc.WriteToken(jsontext.ArrayStart); err != nil {
		return err
	}
	for _, val := range arr {
		if err := marshalValueAny(enc, val, mo); err != nil {
			return err
		}
	}
	if err := enc.WriteToken(jsontext.ArrayEnd); err != nil {
		return err
	}
	return nil
}

func unmarshalArrayAny(dec *jsontext.Decoder, uo *jsonopts.Struct) ([]any, error) {
	tok, err := dec.ReadToken()
	if err != nil {
		return nil, err
	}
	k := tok.Kind()
	switch k {
	case 'n':
		return nil, nil
	case '[':
		arr := []any{}
		for dec.PeekKind() != ']' {
			val, err := unmarshalValueAny(dec, uo)
			arr = append(arr, val)
			if err != nil {
				return arr, err
			}
		}
		if _, err := dec.ReadToken(); err != nil {
			return arr, err
		}
		return arr, nil
	}
	return nil, &SemanticError{action: "unmarshal", JSONKind: k, GoType: sliceAnyType}
}
