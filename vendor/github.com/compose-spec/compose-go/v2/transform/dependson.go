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

	"github.com/compose-spec/compose-go/v2/tree"
)

func transformDependsOn(data any, p tree.Path, _ bool) (any, error) {
	switch v := data.(type) {
	case map[string]any:
		for i, e := range v {
			d, ok := e.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%s.%s: unsupported value %s", p, i, v)
			}
			if _, ok := d["condition"]; !ok {
				d["condition"] = "service_started"
			}
			if _, ok := d["required"]; !ok {
				d["required"] = true
			}
		}
		return v, nil
	case []any:
		d := map[string]any{}
		for _, k := range v {
			d[k.(string)] = map[string]any{
				"condition": "service_started",
				"required":  true,
			}
		}
		return d, nil
	default:
		return data, fmt.Errorf("%s: invalid type %T for depend_on", p, v)
	}
}
