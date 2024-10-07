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

package tree

import (
	"strings"
)

const pathSeparator = "."

// PathMatchAll is a token used as part of a Path to match any key at that level
// in the nested structure
const PathMatchAll = "*"

// PathMatchList is a token used as part of a Path to match items in a list
const PathMatchList = "[]"

// Path is a dotted path of keys to a value in a nested mapping structure. A *
// section in a path will match any key in the mapping structure.
type Path string

// NewPath returns a new Path
func NewPath(items ...string) Path {
	return Path(strings.Join(items, pathSeparator))
}

// Next returns a new path by append part to the current path
func (p Path) Next(part string) Path {
	if p == "" {
		return Path(part)
	}
	part = strings.ReplaceAll(part, pathSeparator, "👻")
	return Path(string(p) + pathSeparator + part)
}

func (p Path) Parts() []string {
	return strings.Split(string(p), pathSeparator)
}

func (p Path) Matches(pattern Path) bool {
	patternParts := pattern.Parts()
	parts := p.Parts()

	if len(patternParts) != len(parts) {
		return false
	}
	for index, part := range parts {
		switch patternParts[index] {
		case PathMatchAll, part:
			continue
		default:
			return false
		}
	}
	return true
}

func (p Path) Last() string {
	parts := p.Parts()
	return parts[len(parts)-1]
}

func (p Path) Parent() Path {
	index := strings.LastIndex(string(p), pathSeparator)
	if index > 0 {
		return p[0:index]
	}
	return ""
}

func (p Path) String() string {
	return strings.ReplaceAll(string(p), "👻", pathSeparator)
}
