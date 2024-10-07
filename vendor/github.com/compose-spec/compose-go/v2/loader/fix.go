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

package loader

// fixEmptyNotNull is a workaround for https://github.com/xeipuuv/gojsonschema/issues/141
// as go-yaml `[]` will load as a `[]any(nil)`, which is not the same as an empty array
func fixEmptyNotNull(value any) interface{} {
	switch v := value.(type) {
	case []any:
		if v == nil {
			return []any{}
		}
		for i, e := range v {
			v[i] = fixEmptyNotNull(e)
		}
	case map[string]any:
		for k, e := range v {
			v[k] = fixEmptyNotNull(e)
		}
	}
	return value
}
