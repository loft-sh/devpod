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

import (
	"fmt"
	"sort"
	"strings"
)

// MappingWithEquals is a mapping type that can be converted from a list of
// key[=value] strings.
// For the key with an empty value (`key=`), the mapped value is set to a pointer to `""`.
// For the key without value (`key`), the mapped value is set to nil.
type MappingWithEquals map[string]*string

// NewMappingWithEquals build a new Mapping from a set of KEY=VALUE strings
func NewMappingWithEquals(values []string) MappingWithEquals {
	mapping := MappingWithEquals{}
	for _, env := range values {
		tokens := strings.SplitN(env, "=", 2)
		if len(tokens) > 1 {
			mapping[tokens[0]] = &tokens[1]
		} else {
			mapping[env] = nil
		}
	}
	return mapping
}

// OverrideBy update MappingWithEquals with values from another MappingWithEquals
func (m MappingWithEquals) OverrideBy(other MappingWithEquals) MappingWithEquals {
	for k, v := range other {
		m[k] = v
	}
	return m
}

// Resolve update a MappingWithEquals for keys without value (`key`, but not `key=`)
func (m MappingWithEquals) Resolve(lookupFn func(string) (string, bool)) MappingWithEquals {
	for k, v := range m {
		if v == nil {
			if value, ok := lookupFn(k); ok {
				m[k] = &value
			}
		}
	}
	return m
}

// RemoveEmpty excludes keys that are not associated with a value
func (m MappingWithEquals) RemoveEmpty() MappingWithEquals {
	for k, v := range m {
		if v == nil {
			delete(m, k)
		}
	}
	return m
}

func (m *MappingWithEquals) DecodeMapstructure(value interface{}) error {
	switch v := value.(type) {
	case map[string]interface{}:
		mapping := make(MappingWithEquals, len(v))
		for k, e := range v {
			mapping[k] = mappingValue(e)
		}
		*m = mapping
	case []interface{}:
		mapping := make(MappingWithEquals, len(v))
		for _, s := range v {
			k, e, ok := strings.Cut(fmt.Sprint(s), "=")
			if !ok {
				mapping[k] = nil
			} else {
				mapping[k] = mappingValue(e)
			}
		}
		*m = mapping
	default:
		return fmt.Errorf("unexpected value type %T for mapping", value)
	}
	return nil
}

// label value can be a string | number | boolean | null
func mappingValue(e interface{}) *string {
	if e == nil {
		return nil
	}
	switch v := e.(type) {
	case string:
		return &v
	default:
		s := fmt.Sprint(v)
		return &s
	}
}

// Mapping is a mapping type that can be converted from a list of
// key[=value] strings.
// For the key with an empty value (`key=`), or key without value (`key`), the
// mapped value is set to an empty string `""`.
type Mapping map[string]string

// NewMapping build a new Mapping from a set of KEY=VALUE strings
func NewMapping(values []string) Mapping {
	mapping := Mapping{}
	for _, value := range values {
		parts := strings.SplitN(value, "=", 2)
		key := parts[0]
		switch {
		case len(parts) == 1:
			mapping[key] = ""
		default:
			mapping[key] = parts[1]
		}
	}
	return mapping
}

// convert values into a set of KEY=VALUE strings
func (m Mapping) Values() []string {
	values := make([]string, 0, len(m))
	for k, v := range m {
		values = append(values, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(values)
	return values
}

// ToMappingWithEquals converts Mapping into a MappingWithEquals with pointer references
func (m Mapping) ToMappingWithEquals() MappingWithEquals {
	mapping := MappingWithEquals{}
	for k, v := range m {
		v := v
		mapping[k] = &v
	}
	return mapping
}

func (m Mapping) Resolve(s string) (string, bool) {
	v, ok := m[s]
	return v, ok
}

func (m Mapping) Clone() Mapping {
	clone := Mapping{}
	for k, v := range m {
		clone[k] = v
	}
	return clone
}

// Merge adds all values from second mapping which are not already defined
func (m Mapping) Merge(o Mapping) Mapping {
	for k, v := range o {
		if _, set := m[k]; !set {
			m[k] = v
		}
	}
	return m
}

func (m *Mapping) DecodeMapstructure(value interface{}) error {
	switch v := value.(type) {
	case map[string]interface{}:
		mapping := make(Mapping, len(v))
		for k, e := range v {
			if e == nil {
				e = ""
			}
			mapping[k] = fmt.Sprint(e)
		}
		*m = mapping
	case []interface{}:
		*m = decodeMapping(v, "=")
	default:
		return fmt.Errorf("unexpected value type %T for mapping", value)
	}
	return nil
}

// Generate a mapping by splitting strings at any of seps, which will be tried
// in-order for each input string. (For example, to allow the preferred 'host=ip'
// in 'extra_hosts', as well as 'host:ip' for backwards compatibility.)
func decodeMapping(v []interface{}, seps ...string) map[string]string {
	mapping := make(Mapping, len(v))
	for _, s := range v {
		for i, sep := range seps {
			k, e, ok := strings.Cut(fmt.Sprint(s), sep)
			if ok {
				// Mapping found with this separator, stop here.
				mapping[k] = e
				break
			} else if i == len(seps)-1 {
				// No more separators to try, map to empty string.
				mapping[k] = ""
			}
		}
	}
	return mapping
}
