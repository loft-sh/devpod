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

package template

import (
	"regexp"
	"strings"
)

type Variable struct {
	Name          string
	DefaultValue  string
	PresenceValue string
	Required      bool
}

// ExtractVariables returns a map of all the variables defined in the specified
// compose file (dict representation) and their default value if any.
func ExtractVariables(configDict map[string]interface{}, pattern *regexp.Regexp) map[string]Variable {
	if pattern == nil {
		pattern = DefaultPattern
	}
	return recurseExtract(configDict, pattern)
}

func recurseExtract(value interface{}, pattern *regexp.Regexp) map[string]Variable {
	m := map[string]Variable{}

	switch value := value.(type) {
	case string:
		if values, is := extractVariable(value, pattern); is {
			for _, v := range values {
				m[v.Name] = v
			}
		}
	case map[string]interface{}:
		for _, elem := range value {
			submap := recurseExtract(elem, pattern)
			for key, value := range submap {
				m[key] = value
			}
		}

	case []interface{}:
		for _, elem := range value {
			if values, is := extractVariable(elem, pattern); is {
				for _, v := range values {
					m[v.Name] = v
				}
			}
		}
	}

	return m
}

func extractVariable(value interface{}, pattern *regexp.Regexp) ([]Variable, bool) {
	sValue, ok := value.(string)
	if !ok {
		return []Variable{}, false
	}
	matches := pattern.FindAllStringSubmatch(sValue, -1)
	if len(matches) == 0 {
		return []Variable{}, false
	}
	values := []Variable{}
	for _, match := range matches {
		groups := matchGroups(match, pattern)
		if escaped := groups[groupEscaped]; escaped != "" {
			continue
		}
		val := groups[groupNamed]
		if val == "" {
			val = groups[groupBraced]
			s := match[0]
			i := getFirstBraceClosingIndex(s)
			if i > 0 {
				val = s[2:i]
				if len(s) > i {
					if v, b := extractVariable(s[i+1:], pattern); b {
						values = append(values, v...)
					}
				}
			}
		}
		name := val
		var defaultValue string
		var presenceValue string
		var required bool
		i := strings.IndexFunc(val, func(r rune) bool {
			if r >= 'a' && r <= 'z' {
				return false
			}
			if r >= 'A' && r <= 'Z' {
				return false
			}
			if r == '_' {
				return false
			}
			return true
		})

		if i > 0 {
			name = val[:i]
			rest := val[i:]
			switch {
			case strings.HasPrefix(rest, ":?"):
				required = true
			case strings.HasPrefix(rest, "?"):
				required = true
			case strings.HasPrefix(rest, ":-"):
				defaultValue = rest[2:]
			case strings.HasPrefix(rest, "-"):
				defaultValue = rest[1:]
			case strings.HasPrefix(rest, ":+"):
				presenceValue = rest[2:]
			case strings.HasPrefix(rest, "+"):
				presenceValue = rest[1:]
			}
		}

		values = append(values, Variable{
			Name:          name,
			DefaultValue:  defaultValue,
			PresenceValue: presenceValue,
			Required:      required,
		})

		if defaultValue != "" {
			if v, b := extractVariable(defaultValue, pattern); b {
				values = append(values, v...)
			}
		}
		if presenceValue != "" {
			if v, b := extractVariable(presenceValue, pattern); b {
				values = append(values, v...)
			}
		}
	}
	return values, len(values) > 0
}
