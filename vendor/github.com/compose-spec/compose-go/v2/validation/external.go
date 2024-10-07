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

package validation

import (
	"fmt"
	"strings"

	"github.com/compose-spec/compose-go/v2/consts"
	"github.com/compose-spec/compose-go/v2/tree"
)

func checkExternal(v map[string]any, p tree.Path) error {
	b, ok := v["external"]
	if !ok {
		return nil
	}
	if !b.(bool) {
		return nil
	}

	for k := range v {
		switch k {
		case "name", "external", consts.Extensions:
			continue
		default:
			if strings.HasPrefix(k, "x-") {
				// custom extension, ignored
				continue
			}
			return fmt.Errorf("%s: conflicting parameters \"external\" and %q specified", p, k)
		}
	}
	return nil
}
