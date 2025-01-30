// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"encoding"
	"errors"
	"reflect"

	"github.com/go-json-experiment/json/internal/jsonflags"
	"github.com/go-json-experiment/json/internal/jsonopts"
	"github.com/go-json-experiment/json/internal/jsonwire"
	"github.com/go-json-experiment/json/jsontext"
)

// Interfaces for custom serialization.
var (
	jsonMarshalerV1Type   = reflect.TypeOf((*MarshalerV1)(nil)).Elem()
	jsonMarshalerV2Type   = reflect.TypeOf((*MarshalerV2)(nil)).Elem()
	jsonUnmarshalerV1Type = reflect.TypeOf((*UnmarshalerV1)(nil)).Elem()
	jsonUnmarshalerV2Type = reflect.TypeOf((*UnmarshalerV2)(nil)).Elem()
	textAppenderType      = reflect.TypeOf((*encodingTextAppender)(nil)).Elem()
	textMarshalerType     = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	textUnmarshalerType   = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

	// TODO(https://go.dev/issue/62384): Use encoding.TextAppender instead of this hack.
	// This exists for now to provide performance benefits to netip types.
	// There is no semantic difference with this change.
	appenderToType = reflect.TypeOf((*interface{ AppendTo([]byte) []byte })(nil)).Elem()
)

// TODO(https://go.dev/issue/62384): Use encoding.TextAppender instead
// and document public support for this method in json.Marshal.
type encodingTextAppender interface {
	AppendText(b []byte) ([]byte, error)
}

// MarshalerV1 is implemented by types that can marshal themselves.
// It is recommended that types implement [MarshalerV2] unless the implementation
// is trying to avoid a hard dependency on the "jsontext" package.
//
// It is recommended that implementations return a buffer that is safe
// for the caller to retain and potentially mutate.
type MarshalerV1 interface {
	MarshalJSON() ([]byte, error)
}

// MarshalerV2 is implemented by types that can marshal themselves.
// It is recommended that types implement MarshalerV2 instead of [MarshalerV1]
// since this is both more performant and flexible.
// If a type implements both MarshalerV1 and MarshalerV2,
// then MarshalerV2 takes precedence. In such a case, both implementations
// should aim to have equivalent behavior for the default marshal options.
//
// The implementation must write only one JSON value to the Encoder and
// must not retain the pointer to [jsontext.Encoder] or the [Options] value.
type MarshalerV2 interface {
	MarshalJSONV2(*jsontext.Encoder, Options) error

	// TODO: Should users call the MarshalEncode function or
	// should/can they call this method directly? Does it matter?
}

// UnmarshalerV1 is implemented by types that can unmarshal themselves.
// It is recommended that types implement [UnmarshalerV2] unless the implementation
// is trying to avoid a hard dependency on the "jsontext" package.
//
// The input can be assumed to be a valid encoding of a JSON value
// if called from unmarshal functionality in this package.
// UnmarshalJSON must copy the JSON data if it is retained after returning.
// It is recommended that UnmarshalJSON implement merge semantics when
// unmarshaling into a pre-populated value.
//
// Implementations must not retain or mutate the input []byte.
type UnmarshalerV1 interface {
	UnmarshalJSON([]byte) error
}

// UnmarshalerV2 is implemented by types that can unmarshal themselves.
// It is recommended that types implement UnmarshalerV2 instead of [UnmarshalerV1]
// since this is both more performant and flexible.
// If a type implements both UnmarshalerV1 and UnmarshalerV2,
// then UnmarshalerV2 takes precedence. In such a case, both implementations
// should aim to have equivalent behavior for the default unmarshal options.
//
// The implementation must read only one JSON value from the Decoder.
// It is recommended that UnmarshalJSONV2 implement merge semantics when
// unmarshaling into a pre-populated value.
//
// Implementations must not retain the pointer to [jsontext.Decoder] or
// the [Options] value.
type UnmarshalerV2 interface {
	UnmarshalJSONV2(*jsontext.Decoder, Options) error

	// TODO: Should users call the UnmarshalDecode function or
	// should/can they call this method directly? Does it matter?
}

func makeMethodArshaler(fncs *arshaler, t reflect.Type) *arshaler {
	// Avoid injecting method arshaler on the pointer or interface version
	// to avoid ever calling the method on a nil pointer or interface receiver.
	// Let it be injected on the value receiver (which is always addressable).
	if t.Kind() == reflect.Pointer || t.Kind() == reflect.Interface {
		return fncs
	}

	// Handle custom marshaler.
	switch which := implementsWhich(t, jsonMarshalerV2Type, jsonMarshalerV1Type, textAppenderType, textMarshalerType); which {
	case jsonMarshalerV2Type:
		fncs.nonDefault = true
		fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
			xe := export.Encoder(enc)
			prevDepth, prevLength := xe.Tokens.DepthLength()
			xe.Flags.Set(jsonflags.WithinArshalCall | 1)
			err := va.Addr().Interface().(MarshalerV2).MarshalJSONV2(enc, mo)
			xe.Flags.Set(jsonflags.WithinArshalCall | 0)
			currDepth, currLength := xe.Tokens.DepthLength()
			if (prevDepth != currDepth || prevLength+1 != currLength) && err == nil {
				err = errors.New("must write exactly one JSON value")
			}
			if err != nil {
				err = wrapSkipFunc(err, "marshal method")
				// TODO: Avoid wrapping semantic or I/O errors.
				return &SemanticError{action: "marshal", GoType: t, Err: err}
			}
			return nil
		}
	case jsonMarshalerV1Type:
		fncs.nonDefault = true
		fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
			marshaler := va.Addr().Interface().(MarshalerV1)
			val, err := marshaler.MarshalJSON()
			if err != nil {
				err = wrapSkipFunc(err, "marshal method")
				// TODO: Avoid wrapping semantic errors.
				return &SemanticError{action: "marshal", GoType: t, Err: err}
			}
			if err := enc.WriteValue(val); err != nil {
				// TODO: Avoid wrapping semantic or I/O errors.
				return &SemanticError{action: "marshal", JSONKind: jsontext.Value(val).Kind(), GoType: t, Err: err}
			}
			return nil
		}
	case textAppenderType:
		fncs.nonDefault = true
		fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) (err error) {
			appender := va.Addr().Interface().(encodingTextAppender)
			if err := export.Encoder(enc).AppendRaw('"', false, appender.AppendText); err != nil {
				// TODO: Avoid wrapping semantic, syntactic, or I/O errors.
				err = wrapSkipFunc(err, "append method")
				return &SemanticError{action: "marshal", JSONKind: '"', GoType: t, Err: err}
			}
			return nil
		}
	case textMarshalerType:
		fncs.nonDefault = true
		fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
			marshaler := va.Addr().Interface().(encoding.TextMarshaler)
			if err := export.Encoder(enc).AppendRaw('"', false, func(b []byte) ([]byte, error) {
				b2, err := marshaler.MarshalText()
				return append(b, b2...), err
			}); err != nil {
				// TODO: Avoid wrapping semantic, syntactic, or I/O errors.
				err = wrapSkipFunc(err, "marshal method")
				return &SemanticError{action: "marshal", JSONKind: '"', GoType: t, Err: err}
			}
			return nil
		}
		// TODO(https://go.dev/issue/62384): Rely on encoding.TextAppender instead.
		if implementsWhich(t, appenderToType) != nil && t.PkgPath() == "net/netip" {
			fncs.marshal = func(enc *jsontext.Encoder, va addressableValue, mo *jsonopts.Struct) error {
				appender := va.Addr().Interface().(interface{ AppendTo([]byte) []byte })
				if err := export.Encoder(enc).AppendRaw('"', false, func(b []byte) ([]byte, error) {
					return appender.AppendTo(b), nil
				}); err != nil {
					// TODO: Avoid wrapping semantic, syntactic, or I/O errors.
					err = wrapSkipFunc(err, "append method")
					return &SemanticError{action: "marshal", JSONKind: '"', GoType: t, Err: err}
				}
				return nil
			}
		}
	}

	// Handle custom unmarshaler.
	switch which := implementsWhich(t, jsonUnmarshalerV2Type, jsonUnmarshalerV1Type, textUnmarshalerType); which {
	case jsonUnmarshalerV2Type:
		fncs.nonDefault = true
		fncs.unmarshal = func(dec *jsontext.Decoder, va addressableValue, uo *jsonopts.Struct) error {
			xd := export.Decoder(dec)
			prevDepth, prevLength := xd.Tokens.DepthLength()
			xd.Flags.Set(jsonflags.WithinArshalCall | 1)
			err := va.Addr().Interface().(UnmarshalerV2).UnmarshalJSONV2(dec, uo)
			xd.Flags.Set(jsonflags.WithinArshalCall | 0)
			currDepth, currLength := xd.Tokens.DepthLength()
			if (prevDepth != currDepth || prevLength+1 != currLength) && err == nil {
				err = errors.New("must read exactly one JSON value")
			}
			if err != nil {
				err = wrapSkipFunc(err, "unmarshal method")
				// TODO: Avoid wrapping semantic, syntactic, or I/O errors.
				return &SemanticError{action: "unmarshal", GoType: t, Err: err}
			}
			return nil
		}
	case jsonUnmarshalerV1Type:
		fncs.nonDefault = true
		fncs.unmarshal = func(dec *jsontext.Decoder, va addressableValue, uo *jsonopts.Struct) error {
			val, err := dec.ReadValue()
			if err != nil {
				return err // must be a syntactic or I/O error
			}
			unmarshaler := va.Addr().Interface().(UnmarshalerV1)
			if err := unmarshaler.UnmarshalJSON(val); err != nil {
				err = wrapSkipFunc(err, "unmarshal method")
				// TODO: Avoid wrapping semantic, syntactic, or I/O errors.
				return &SemanticError{action: "unmarshal", JSONKind: val.Kind(), GoType: t, Err: err}
			}
			return nil
		}
	case textUnmarshalerType:
		fncs.nonDefault = true
		fncs.unmarshal = func(dec *jsontext.Decoder, va addressableValue, uo *jsonopts.Struct) error {
			xd := export.Decoder(dec)
			var flags jsonwire.ValueFlags
			val, err := xd.ReadValue(&flags)
			if err != nil {
				return err // must be a syntactic or I/O error
			}
			if val.Kind() != '"' {
				err = errors.New("JSON value must be string type")
				return &SemanticError{action: "unmarshal", JSONKind: val.Kind(), GoType: t, Err: err}
			}
			s := jsonwire.UnquoteMayCopy(val, flags.IsVerbatim())
			unmarshaler := va.Addr().Interface().(encoding.TextUnmarshaler)
			if err := unmarshaler.UnmarshalText(s); err != nil {
				err = wrapSkipFunc(err, "unmarshal method")
				// TODO: Avoid wrapping semantic, syntactic, or I/O errors.
				return &SemanticError{action: "unmarshal", JSONKind: val.Kind(), GoType: t, Err: err}
			}
			return nil
		}
	}

	return fncs
}

// implementsWhich is like t.Implements(ifaceType) for a list of interfaces,
// but checks whether either t or reflect.PointerTo(t) implements the interface.
func implementsWhich(t reflect.Type, ifaceTypes ...reflect.Type) (which reflect.Type) {
	for _, ifaceType := range ifaceTypes {
		if t.Implements(ifaceType) || reflect.PointerTo(t).Implements(ifaceType) {
			return ifaceType
		}
	}
	return nil
}
