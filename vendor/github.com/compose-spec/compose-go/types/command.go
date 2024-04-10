/*
   Copyright 2020 The Compose Specification Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package types

import "github.com/mattn/go-shellwords"

// ShellCommand is a string or list of string args.
//
// When marshaled to YAML, nil command fields will be omitted if `omitempty`
// is specified as a struct tag. Explicitly empty commands (i.e. `[]` or
// empty string will serialize to an empty array (`[]`).
//
// When marshaled to JSON, the `omitempty` struct must NOT be specified.
// If the command field is nil, it will be serialized as `null`.
// Explicitly empty commands (i.e. `[]` or empty string) will serialize to
// an empty array (`[]`).
//
// The distinction between nil and explicitly empty is important to distinguish
// between an unset value and a provided, but empty, value, which should be
// preserved so that it can override any base value (e.g. container entrypoint).
//
// The different semantics between YAML and JSON are due to limitations with
// JSON marshaling + `omitempty` in the Go stdlib, while gopkg.in/yaml.v3 gives
// us more flexibility via the yaml.IsZeroer interface.
//
// In the future, it might make sense to make fields of this type be
// `*ShellCommand` to avoid this situation, but that would constitute a
// breaking change.
type ShellCommand []string

// IsZero returns true if the slice is nil.
//
// Empty (but non-nil) slices are NOT considered zero values.
func (s ShellCommand) IsZero() bool {
	// we do NOT want len(s) == 0, ONLY explicitly nil
	return s == nil
}

// MarshalYAML returns nil (which will be serialized as `null`) for nil slices
// and delegates to the standard marshaller behavior otherwise.
//
// NOTE: Typically the nil case here is not hit because IsZero has already
// short-circuited marshalling, but this ensures that the type serializes
// accurately if the `omitempty` struct tag is omitted/forgotten.
//
// A similar MarshalJSON() implementation is not needed because the Go stdlib
// already serializes nil slices to `null`, whereas gopkg.in/yaml.v3 by default
// serializes nil slices to `[]`.
func (s ShellCommand) MarshalYAML() (interface{}, error) {
	if s == nil {
		return nil, nil
	}
	return []string(s), nil
}

func (s *ShellCommand) DecodeMapstructure(value interface{}) error {
	switch v := value.(type) {
	case string:
		cmd, err := shellwords.Parse(v)
		if err != nil {
			return err
		}
		*s = cmd
	case []interface{}:
		cmd := make([]string, len(v))
		for i, s := range v {
			cmd[i] = s.(string)
		}
		*s = cmd
	}
	return nil
}
