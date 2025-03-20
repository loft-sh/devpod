// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/go-json-experiment/json/jsontext"
)

const errorPrefix = "json: "

// SemanticError describes an error determining the meaning
// of JSON data as Go data or vice-versa.
//
// The contents of this error as produced by this package may change over time.
type SemanticError struct {
	requireKeyedLiterals
	nonComparable

	action string // either "marshal" or "unmarshal"

	// ByteOffset indicates that an error occurred after this byte offset.
	ByteOffset int64
	// JSONPointer indicates that an error occurred within this JSON value
	// as indicated using the JSON Pointer notation (see RFC 6901).
	JSONPointer string

	// JSONKind is the JSON kind that could not be handled.
	JSONKind jsontext.Kind // may be zero if unknown
	// GoType is the Go type that could not be handled.
	GoType reflect.Type // may be nil if unknown

	// Err is the underlying error.
	Err error // may be nil
}

func (e *SemanticError) Error() string {
	var sb strings.Builder
	sb.WriteString(errorPrefix)

	// Hyrum-proof the error message by deliberately switching between
	// two equivalent renderings of the same error message.
	// The randomization is tied to the Hyrum-proofing already applied
	// on map iteration in Go.
	for phrase := range map[string]struct{}{"cannot": {}, "unable to": {}} {
		sb.WriteString(phrase)
		break // use whichever phrase we get in the first iteration
	}

	// Format action.
	var preposition string
	switch e.action {
	case "marshal":
		sb.WriteString(" marshal")
		preposition = " from"
	case "unmarshal":
		sb.WriteString(" unmarshal")
		preposition = " into"
	default:
		sb.WriteString(" handle")
		preposition = " with"
	}

	// Format JSON kind.
	var omitPreposition bool
	switch e.JSONKind {
	case 'n':
		sb.WriteString(" JSON null")
	case 'f', 't':
		sb.WriteString(" JSON boolean")
	case '"':
		sb.WriteString(" JSON string")
	case '0':
		sb.WriteString(" JSON number")
	case '{', '}':
		sb.WriteString(" JSON object")
	case '[', ']':
		sb.WriteString(" JSON array")
	default:
		omitPreposition = true
	}

	// Format Go type.
	if e.GoType != nil {
		if !omitPreposition {
			sb.WriteString(preposition)
		}
		sb.WriteString(" Go value of type ")
		sb.WriteString(e.GoType.String())
	}

	// Format where.
	switch {
	case e.JSONPointer != "":
		sb.WriteString(" within JSON value at ")
		sb.WriteString(strconv.Quote(e.JSONPointer))
	case e.ByteOffset > 0:
		sb.WriteString(" after byte offset ")
		sb.WriteString(strconv.FormatInt(e.ByteOffset, 10))
	}

	// Format underlying error.
	if e.Err != nil {
		sb.WriteString(": ")
		sb.WriteString(e.Err.Error())
	}

	return sb.String()
}
func (e *SemanticError) Unwrap() error {
	return e.Err
}

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
