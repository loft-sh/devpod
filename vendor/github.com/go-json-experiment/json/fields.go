// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-json-experiment/json/internal/jsonflags"
	"github.com/go-json-experiment/json/internal/jsonwire"
)

var errIgnoredField = errors.New("ignored field")

type isZeroer interface {
	IsZero() bool
}

var isZeroerType = reflect.TypeOf((*isZeroer)(nil)).Elem()

type structFields struct {
	flattened       []structField // listed in depth-first ordering
	byActualName    map[string]*structField
	byFoldedName    map[string][]*structField
	inlinedFallback *structField
}

// lookupByFoldedName looks up name by a case-insensitive match
// that also ignores the presence of dashes and underscores.
func (fs *structFields) lookupByFoldedName(name []byte) []*structField {
	return fs.byFoldedName[string(foldName(name))]
}

type structField struct {
	id      int   // unique numeric ID in breadth-first ordering
	index   []int // index into a struct according to reflect.Type.FieldByIndex
	typ     reflect.Type
	fncs    *arshaler
	isZero  func(addressableValue) bool
	isEmpty func(addressableValue) bool
	fieldOptions
}

func makeStructFields(root reflect.Type) (structFields, *SemanticError) {
	// Setup a queue for a breath-first search.
	var queueIndex int
	type queueEntry struct {
		typ           reflect.Type
		index         []int
		visitChildren bool // whether to recursively visit inlined field in this struct
	}
	queue := []queueEntry{{root, nil, true}}
	seen := map[reflect.Type]bool{root: true}

	// Perform a breadth-first search over all reachable fields.
	// This ensures that len(f.index) will be monotonically increasing.
	var allFields, inlinedFallbacks []structField
	for queueIndex < len(queue) {
		qe := queue[queueIndex]
		queueIndex++

		t := qe.typ
		inlinedFallbackIndex := -1         // index of last inlined fallback field in current struct
		namesIndex := make(map[string]int) // index of each field with a given JSON object name in current struct
		var hasAnyJSONTag bool             // whether any Go struct field has a `json` tag
		var hasAnyJSONField bool           // whether any JSON serializable fields exist in current struct
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			_, hasTag := sf.Tag.Lookup("json")
			hasAnyJSONTag = hasAnyJSONTag || hasTag
			options, ignored, err := parseFieldOptions(sf)
			if err != nil {
				return structFields{}, &SemanticError{GoType: t, Err: err}
			} else if ignored {
				continue
			}
			hasAnyJSONField = true
			f := structField{
				// Allocate a new slice (len=N+1) to hold both
				// the parent index (len=N) and the current index (len=1).
				// Do this to avoid clobbering the memory of the parent index.
				index:        append(append(make([]int, 0, len(qe.index)+1), qe.index...), i),
				typ:          sf.Type,
				fieldOptions: options,
			}
			if sf.Anonymous && !f.hasName {
				f.inline = true // implied by use of Go embedding without an explicit name
			}
			if f.inline || f.unknown {
				// Handle an inlined field that serializes to/from
				// zero or more JSON object members.

				if f.inline && f.unknown {
					err := fmt.Errorf("Go struct field %s cannot have both `inline` and `unknown` specified", sf.Name)
					return structFields{}, &SemanticError{GoType: t, Err: err}
				}
				switch f.fieldOptions {
				case fieldOptions{name: f.name, quotedName: f.quotedName, inline: true}:
				case fieldOptions{name: f.name, quotedName: f.quotedName, unknown: true}:
				default:
					err := fmt.Errorf("Go struct field %s cannot have any options other than `inline` or `unknown` specified", sf.Name)
					return structFields{}, &SemanticError{GoType: t, Err: err}
				}

				// Unwrap one level of pointer indirection similar to how Go
				// only allows embedding either T or *T, but not **T.
				tf := f.typ
				if tf.Kind() == reflect.Pointer && tf.Name() == "" {
					tf = tf.Elem()
				}
				// Reject any types with custom serialization otherwise
				// it becomes impossible to know what sub-fields to inline.
				if which := implementsWhich(tf,
					jsonMarshalerV2Type, jsonMarshalerV1Type, textMarshalerType,
					jsonUnmarshalerV2Type, jsonUnmarshalerV1Type, textUnmarshalerType,
				); which != nil && tf != jsontextValueType {
					err := fmt.Errorf("inlined Go struct field %s of type %s must not implement JSON marshal or unmarshal methods", sf.Name, tf)
					return structFields{}, &SemanticError{GoType: t, Err: err}
				}

				// Handle an inlined field that serializes to/from
				// a finite number of JSON object members backed by a Go struct.
				if tf.Kind() == reflect.Struct {
					if f.unknown {
						err := fmt.Errorf("inlined Go struct field %s of type %s with `unknown` tag must be a Go map of string key or a jsontext.Value", sf.Name, tf)
						return structFields{}, &SemanticError{GoType: t, Err: err}
					}
					if qe.visitChildren {
						queue = append(queue, queueEntry{tf, f.index, !seen[tf]})
					}
					seen[tf] = true
					continue
				}

				// Handle an inlined field that serializes to/from any number of
				// JSON object members back by a Go map or jsontext.Value.
				switch {
				case tf == jsontextValueType:
					f.fncs = nil // specially handled in arshal_inlined.go
				case tf.Kind() == reflect.Map && tf.Key() == stringType:
					f.fncs = lookupArshaler(tf.Elem())
				default:
					err := fmt.Errorf("inlined Go struct field %s of type %s must be a Go struct, Go map of string key, or jsontext.Value", sf.Name, tf)
					return structFields{}, &SemanticError{GoType: t, Err: err}
				}

				// Reject multiple inlined fallback fields within the same struct.
				if inlinedFallbackIndex >= 0 {
					err := fmt.Errorf("inlined Go struct fields %s and %s cannot both be a Go map or jsontext.Value", t.Field(inlinedFallbackIndex).Name, sf.Name)
					return structFields{}, &SemanticError{GoType: t, Err: err}
				}
				inlinedFallbackIndex = i

				inlinedFallbacks = append(inlinedFallbacks, f)
			} else {
				// Handle normal Go struct field that serializes to/from
				// a single JSON object member.

				// Provide a function that uses a type's IsZero method.
				switch {
				case sf.Type.Kind() == reflect.Interface && sf.Type.Implements(isZeroerType):
					f.isZero = func(va addressableValue) bool {
						// Avoid panics calling IsZero on a nil interface or
						// non-nil interface with nil pointer.
						return va.IsNil() || (va.Elem().Kind() == reflect.Pointer && va.Elem().IsNil()) || va.Interface().(isZeroer).IsZero()
					}
				case sf.Type.Kind() == reflect.Pointer && sf.Type.Implements(isZeroerType):
					f.isZero = func(va addressableValue) bool {
						// Avoid panics calling IsZero on nil pointer.
						return va.IsNil() || va.Interface().(isZeroer).IsZero()
					}
				case sf.Type.Implements(isZeroerType):
					f.isZero = func(va addressableValue) bool { return va.Interface().(isZeroer).IsZero() }
				case reflect.PointerTo(sf.Type).Implements(isZeroerType):
					f.isZero = func(va addressableValue) bool { return va.Addr().Interface().(isZeroer).IsZero() }
				}

				// Provide a function that can determine whether the value would
				// serialize as an empty JSON value.
				switch sf.Type.Kind() {
				case reflect.String, reflect.Map, reflect.Array, reflect.Slice:
					f.isEmpty = func(va addressableValue) bool { return va.Len() == 0 }
				case reflect.Pointer, reflect.Interface:
					f.isEmpty = func(va addressableValue) bool { return va.IsNil() }
				}

				// Reject user-specified names with invalid UTF-8.
				if !utf8.ValidString(f.name) {
					err := fmt.Errorf("Go struct field %s has JSON object name %q with invalid UTF-8", sf.Name, f.name)
					return structFields{}, &SemanticError{GoType: t, Err: err}
				}
				// Reject multiple fields with same name within the same struct.
				if j, ok := namesIndex[f.name]; ok {
					err := fmt.Errorf("Go struct fields %s and %s conflict over JSON object name %q", t.Field(j).Name, sf.Name, f.name)
					return structFields{}, &SemanticError{GoType: t, Err: err}
				}
				namesIndex[f.name] = i

				f.id = len(allFields)
				f.fncs = lookupArshaler(sf.Type)
				allFields = append(allFields, f)
			}
		}

		// NOTE: New users to the json package are occasionally surprised that
		// unexported fields are ignored. This occurs by necessity due to our
		// inability to directly introspect such fields with Go reflection
		// without the use of unsafe.
		//
		// To reduce friction here, refuse to serialize any Go struct that
		// has no JSON serializable fields, has at least one Go struct field,
		// and does not have any `json` tags present. For example,
		// errors returned by errors.New would fail to serialize.
		isEmptyStruct := t.NumField() == 0
		if !isEmptyStruct && !hasAnyJSONTag && !hasAnyJSONField {
			err := errors.New("Go struct has no exported fields")
			return structFields{}, &SemanticError{GoType: t, Err: err}
		}
	}

	// Sort the fields by exact name (breaking ties by depth and
	// then by presence of an explicitly provided JSON name).
	// Select the dominant field from each set of fields with the same name.
	// If multiple fields have the same name, then the dominant field
	// is the one that exists alone at the shallowest depth,
	// or the one that is uniquely tagged with a JSON name.
	// Otherwise, no dominant field exists for the set.
	flattened := allFields[:0]
	slices.SortFunc(allFields, func(x, y structField) int {
		switch {
		case x.name != y.name:
			return strings.Compare(x.name, y.name)
		case len(x.index) != len(y.index):
			return cmp.Compare(len(x.index), len(y.index))
		case x.hasName && !y.hasName:
			return -1
		case !x.hasName && y.hasName:
			return +1
		default:
			return 0 // TODO(https://go.dev/issue/61643): Compare bools better.
		}
	})
	for len(allFields) > 0 {
		n := 1 // number of fields with the same exact name
		for n < len(allFields) && allFields[n-1].name == allFields[n].name {
			n++
		}
		if n == 1 || len(allFields[0].index) != len(allFields[1].index) || allFields[0].hasName != allFields[1].hasName {
			flattened = append(flattened, allFields[0]) // only keep field if there is a dominant field
		}
		allFields = allFields[n:]
	}

	// Sort the fields according to a breadth-first ordering
	// so that we can re-number IDs with the smallest possible values.
	// This optimizes use of uintSet such that it fits in the 64-entry bit set.
	slices.SortFunc(flattened, func(x, y structField) int {
		return cmp.Compare(x.id, y.id)
	})
	for i := range flattened {
		flattened[i].id = i
	}

	// Sort the fields according to a depth-first ordering
	// as the typical order that fields are marshaled.
	slices.SortFunc(flattened, func(x, y structField) int {
		return slices.Compare(x.index, y.index)
	})

	// Compute the mapping of fields in the byActualName map.
	// Pre-fold all names so that we can lookup folded names quickly.
	fs := structFields{
		flattened:    flattened,
		byActualName: make(map[string]*structField, len(flattened)),
		byFoldedName: make(map[string][]*structField, len(flattened)),
	}
	for i, f := range fs.flattened {
		foldedName := string(foldName([]byte(f.name)))
		fs.byActualName[f.name] = &fs.flattened[i]
		fs.byFoldedName[foldedName] = append(fs.byFoldedName[foldedName], &fs.flattened[i])
	}
	for foldedName, fields := range fs.byFoldedName {
		if len(fields) > 1 {
			// The precedence order for conflicting nocase names
			// is by breadth-first order, rather than depth-first order.
			slices.SortFunc(fields, func(x, y *structField) int {
				return cmp.Compare(x.id, y.id)
			})
			fs.byFoldedName[foldedName] = fields
		}
	}
	if n := len(inlinedFallbacks); n == 1 || (n > 1 && len(inlinedFallbacks[0].index) != len(inlinedFallbacks[1].index)) {
		fs.inlinedFallback = &inlinedFallbacks[0] // dominant inlined fallback field
	}

	return fs, nil
}

// matchFoldedName matches a case-insensitive name depending on the options.
// It assumes that foldName(f.name) == foldName(name).
//
// Case-insensitive matching is used if the `nocase` tag option is specified
// or the MatchCaseInsensitiveNames call option is specified
// (and the `strictcase` tag option is not specified).
// Functionally, the `nocase` and `strictcase` tag options take precedence.
//
// The v1 definition of case-insensitivity operated under strings.EqualFold
// and would strictly compare dashes and underscores,
// while the v2 definition would ignore the presence of dashes and underscores.
// Thus, if the MatchCaseSensitiveDelimiter call option is specified,
// the match is further restricted to using strings.EqualFold.
func (f *structField) matchFoldedName(name []byte, flags *jsonflags.Flags) bool {
	if f.casing == nocase || (flags.Get(jsonflags.MatchCaseInsensitiveNames) && f.casing != strictcase) {
		if !flags.Get(jsonflags.MatchCaseSensitiveDelimiter) || strings.EqualFold(string(name), f.name) {
			return true
		}
	}
	return false
}

const (
	nocase     = 1
	strictcase = 2
)

type fieldOptions struct {
	name       string
	quotedName string // quoted name per RFC 8785, section 3.2.2.2.
	hasName    bool
	casing     int8 // either 0, nocase, or strictcase
	inline     bool
	unknown    bool
	omitzero   bool
	omitempty  bool
	string     bool
	format     string
}

// parseFieldOptions parses the `json` tag in a Go struct field as
// a structured set of options configuring parameters such as
// the JSON member name and other features.
func parseFieldOptions(sf reflect.StructField) (out fieldOptions, ignored bool, err error) {
	tag, hasTag := sf.Tag.Lookup("json")

	// Check whether this field is explicitly ignored.
	if tag == "-" {
		return fieldOptions{}, true, nil
	}

	// Check whether this field is unexported.
	if !sf.IsExported() {
		// In contrast to v1, v2 no longer forwards exported fields from
		// embedded fields of unexported types since Go reflection does not
		// allow the same set of operations that are available in normal cases
		// of purely exported fields.
		// See https://go.dev/issue/21357 and https://go.dev/issue/24153.
		if sf.Anonymous {
			err = firstError(err, fmt.Errorf("embedded Go struct field %s of an unexported type must be explicitly ignored with a `json:\"-\"` tag", sf.Type.Name()))
		}
		// Tag options specified on an unexported field suggests user error.
		if hasTag {
			err = firstError(err, fmt.Errorf("unexported Go struct field %s cannot have non-ignored `json:%q` tag", sf.Name, tag))
		}
		return fieldOptions{}, true, err
	}

	// Determine the JSON member name for this Go field. A user-specified name
	// may be provided as either an identifier or a single-quoted string.
	// The single-quoted string allows arbitrary characters in the name.
	// See https://go.dev/issue/2718 and https://go.dev/issue/3546.
	out.name = sf.Name // always starts with an uppercase character
	if len(tag) > 0 && !strings.HasPrefix(tag, ",") {
		// For better compatibility with v1, accept almost any unescaped name.
		n := len(tag) - len(strings.TrimLeftFunc(tag, func(r rune) bool {
			return !strings.ContainsRune(",\\'\"`", r) // reserve comma, backslash, and quotes
		}))
		opt := tag[:n]
		if n == 0 {
			// Allow a single quoted string for arbitrary names.
			var err2 error
			opt, n, err2 = consumeTagOption(tag)
			if err2 != nil {
				err = firstError(err, fmt.Errorf("Go struct field %s has malformed `json` tag: %v", sf.Name, err2))
			}
		}
		out.hasName = true
		out.name = opt
		tag = tag[n:]
	}
	b, _ := jsonwire.AppendQuote(nil, out.name, &jsonflags.Flags{})
	out.quotedName = string(b)

	// Handle any additional tag options (if any).
	var wasFormat bool
	seenOpts := make(map[string]bool)
	for len(tag) > 0 {
		// Consume comma delimiter.
		if tag[0] != ',' {
			err = firstError(err, fmt.Errorf("Go struct field %s has malformed `json` tag: invalid character %q before next option (expecting ',')", sf.Name, tag[0]))
		} else {
			tag = tag[len(","):]
			if len(tag) == 0 {
				err = firstError(err, fmt.Errorf("Go struct field %s has malformed `json` tag: invalid trailing ',' character", sf.Name))
				break
			}
		}

		// Consume and process the tag option.
		opt, n, err2 := consumeTagOption(tag)
		if err2 != nil {
			err = firstError(err, fmt.Errorf("Go struct field %s has malformed `json` tag: %v", sf.Name, err2))
		}
		rawOpt := tag[:n]
		tag = tag[n:]
		switch {
		case wasFormat:
			err = firstError(err, fmt.Errorf("Go struct field %s has `format` tag option that was not specified last", sf.Name))
		case strings.HasPrefix(rawOpt, "'") && strings.TrimFunc(opt, isLetterOrDigit) == "":
			err = firstError(err, fmt.Errorf("Go struct field %s has unnecessarily quoted appearance of `%s` tag option; specify `%s` instead", sf.Name, rawOpt, opt))
		}
		switch opt {
		case "nocase":
			out.casing |= nocase
		case "strictcase":
			out.casing |= strictcase
		case "inline":
			out.inline = true
		case "unknown":
			out.unknown = true
		case "omitzero":
			out.omitzero = true
		case "omitempty":
			out.omitempty = true
		case "string":
			out.string = true
		case "format":
			if !strings.HasPrefix(tag, ":") {
				err = firstError(err, fmt.Errorf("Go struct field %s is missing value for `format` tag option", sf.Name))
				break
			}
			tag = tag[len(":"):]
			opt, n, err2 := consumeTagOption(tag)
			if err2 != nil {
				err = firstError(err, fmt.Errorf("Go struct field %s has malformed value for `format` tag option: %v", sf.Name, err2))
				break
			}
			tag = tag[n:]
			out.format = opt
			wasFormat = true
		default:
			// Reject keys that resemble one of the supported options.
			// This catches invalid mutants such as "omitEmpty" or "omit_empty".
			normOpt := strings.ReplaceAll(strings.ToLower(opt), "_", "")
			switch normOpt {
			case "nocase", "strictcase", "inline", "unknown", "omitzero", "omitempty", "string", "format":
				err = firstError(err, fmt.Errorf("Go struct field %s has invalid appearance of `%s` tag option; specify `%s` instead", sf.Name, opt, normOpt))
			}

			// NOTE: Everything else is ignored. This does not mean it is
			// forward compatible to insert arbitrary tag options since
			// a future version of this package may understand that tag.
		}

		// Reject duplicates.
		switch {
		case out.casing == nocase|strictcase:
			err = firstError(err, fmt.Errorf("Go struct field %s cannot have both `nocase` and `structcase` tag options", sf.Name))
		case seenOpts[opt]:
			err = firstError(err, fmt.Errorf("Go struct field %s has duplicate appearance of `%s` tag option", sf.Name, rawOpt))
		}
		seenOpts[opt] = true
	}
	return out, false, err
}

func consumeTagOption(in string) (string, int, error) {
	// For legacy compatibility with v1, assume options are comma-separated.
	i := strings.IndexByte(in, ',')
	if i < 0 {
		i = len(in)
	}

	switch r, _ := utf8.DecodeRuneInString(in); {
	// Option as a Go identifier.
	case r == '_' || unicode.IsLetter(r):
		n := len(in) - len(strings.TrimLeftFunc(in, isLetterOrDigit))
		return in[:n], n, nil
	// Option as a single-quoted string.
	case r == '\'':
		// The grammar is nearly identical to a double-quoted Go string literal,
		// but uses single quotes as the terminators. The reason for a custom
		// grammar is because both backtick and double quotes cannot be used
		// verbatim in a struct tag.
		//
		// Convert a single-quoted string to a double-quote string and rely on
		// strconv.Unquote to handle the rest.
		var inEscape bool
		b := []byte{'"'}
		n := len(`'`)
		for len(in) > n {
			r, rn := utf8.DecodeRuneInString(in[n:])
			switch {
			case inEscape:
				if r == '\'' {
					b = b[:len(b)-1] // remove escape character: `\'` => `'`
				}
				inEscape = false
			case r == '\\':
				inEscape = true
			case r == '"':
				b = append(b, '\\') // insert escape character: `"` => `\"`
			case r == '\'':
				b = append(b, '"')
				n += len(`'`)
				out, err := strconv.Unquote(string(b))
				if err != nil {
					return in[:i], i, fmt.Errorf("invalid single-quoted string: %s", in[:n])
				}
				return out, n, nil
			}
			b = append(b, in[n:][:rn]...)
			n += rn
		}
		if n > 10 {
			n = 10 // limit the amount of context printed in the error
		}
		return in[:i], i, fmt.Errorf("single-quoted string not terminated: %s...", in[:n])
	case len(in) == 0:
		return in[:i], i, io.ErrUnexpectedEOF
	default:
		return in[:i], i, fmt.Errorf("invalid character %q at start of option (expecting Unicode letter or single quote)", r)
	}
}

func isLetterOrDigit(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
}
