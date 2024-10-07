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

package transform

import (
	"fmt"
	"strings"

	"github.com/compose-spec/compose-go/v2/tree"
)

func transformKeyValue(data any, p tree.Path, ignoreParseError bool) (any, error) {
	switch v := data.(type) {
	case map[string]any:
		return v, nil
	case []any:
		mapping := map[string]any{}
		for _, e := range v {
			before, after, found := strings.Cut(e.(string), "=")
			if !found {
				if ignoreParseError {
					return data, nil
				}
				return nil, fmt.Errorf("%s: invalid value %s, expected key=value", p, e)
			}
			mapping[before] = after
		}
		return mapping, nil
	default:
		return nil, fmt.Errorf("%s: invalid type %T", p, v)
	}
}
