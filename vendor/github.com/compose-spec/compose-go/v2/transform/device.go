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

func transformDeviceMapping(data any, p tree.Path, ignoreParseError bool) (any, error) {
	switch v := data.(type) {
	case map[string]any:
		return v, nil
	case string:
		src := ""
		dst := ""
		permissions := "rwm"
		arr := strings.Split(v, ":")
		switch len(arr) {
		case 3:
			permissions = arr[2]
			fallthrough
		case 2:
			dst = arr[1]
			fallthrough
		case 1:
			src = arr[0]
		default:
			if !ignoreParseError {
				return nil, fmt.Errorf("confusing device mapping, please use long syntax: %s", v)
			}
		}
		if dst == "" {
			dst = src
		}
		return map[string]any{
			"source":      src,
			"target":      dst,
			"permissions": permissions,
		}, nil
	default:
		return data, fmt.Errorf("%s: invalid type %T for service volume mount", p, v)
	}
}
