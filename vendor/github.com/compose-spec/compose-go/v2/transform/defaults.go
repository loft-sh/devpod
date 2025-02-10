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
	"github.com/compose-spec/compose-go/v2/tree"
)

var defaultValues = map[tree.Path]transformFunc{}

func init() {
	defaultValues["services.*.build"] = defaultBuildContext
	defaultValues["services.*.secrets.*"] = defaultSecretMount
	defaultValues["services.*.ports.*"] = portDefaults
}

// SetDefaultValues transforms a compose model to set default values to missing attributes
func SetDefaultValues(yaml map[string]any) (map[string]any, error) {
	result, err := setDefaults(yaml, tree.NewPath())
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

func setDefaults(data any, p tree.Path) (any, error) {
	for pattern, transformer := range defaultValues {
		if p.Matches(pattern) {
			t, err := transformer(data, p, false)
			if err != nil {
				return nil, err
			}
			return t, nil
		}
	}
	switch v := data.(type) {
	case map[string]any:
		a, err := setDefaultsMapping(v, p)
		if err != nil {
			return a, err
		}
		return v, nil
	case []any:
		a, err := setDefaultsSequence(v, p)
		if err != nil {
			return a, err
		}
		return v, nil
	default:
		return data, nil
	}
}

func setDefaultsSequence(v []any, p tree.Path) ([]any, error) {
	for i, e := range v {
		t, err := setDefaults(e, p.Next("[]"))
		if err != nil {
			return nil, err
		}
		v[i] = t
	}
	return v, nil
}

func setDefaultsMapping(v map[string]any, p tree.Path) (map[string]any, error) {
	for k, e := range v {
		t, err := setDefaults(e, p.Next(k))
		if err != nil {
			return nil, err
		}
		v[k] = t
	}
	return v, nil
}
