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
	"github.com/sirupsen/logrus"
)

func transformMaybeExternal(data any, p tree.Path, ignoreParseError bool) (any, error) {
	if data == nil {
		return nil, nil
	}
	resource, err := transformMapping(data.(map[string]any), p, ignoreParseError)
	if err != nil {
		return nil, err
	}

	if ext, ok := resource["external"]; ok {
		name, named := resource["name"]
		if external, ok := ext.(map[string]any); ok {
			resource["external"] = true
			if extname, extNamed := external["name"]; extNamed {
				logrus.Warnf("%s: external.name is deprecated. Please set name and external: true", p)
				if named && extname != name {
					return nil, fmt.Errorf("%s: name and external.name conflict; only use name", p)
				}
				if !named {
					// adopt (deprecated) external.name if set
					resource["name"] = extname
					return resource, nil
				}
			}
		}
	}

	return resource, nil
}
