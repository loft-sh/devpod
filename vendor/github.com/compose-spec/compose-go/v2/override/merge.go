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

package override

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/compose-spec/compose-go/v2/tree"
	"golang.org/x/exp/slices"
)

// Merge applies overrides to a config model
func Merge(right, left map[string]any) (map[string]any, error) {
	merged, err := mergeYaml(right, left, tree.NewPath())
	if err != nil {
		return nil, err
	}
	return merged.(map[string]any), nil
}

type merger func(any, any, tree.Path) (any, error)

// mergeSpecials defines the custom rules applied by compose when merging yaml trees
var mergeSpecials = map[tree.Path]merger{}

func init() {
	mergeSpecials["networks.*.ipam.config"] = mergeIPAMConfig
	mergeSpecials["networks.*.labels"] = mergeToSequence
	mergeSpecials["volumes.*.labels"] = mergeToSequence
	mergeSpecials["services.*.annotations"] = mergeToSequence
	mergeSpecials["services.*.build"] = mergeBuild
	mergeSpecials["services.*.build.args"] = mergeToSequence
	mergeSpecials["services.*.build.additional_contexts"] = mergeToSequence
	mergeSpecials["services.*.build.extra_hosts"] = mergeToSequence
	mergeSpecials["services.*.build.labels"] = mergeToSequence
	mergeSpecials["services.*.command"] = override
	mergeSpecials["services.*.depends_on"] = mergeDependsOn
	mergeSpecials["services.*.deploy.labels"] = mergeToSequence
	mergeSpecials["services.*.dns"] = mergeToSequence
	mergeSpecials["services.*.dns_opt"] = mergeToSequence
	mergeSpecials["services.*.dns_search"] = mergeToSequence
	mergeSpecials["services.*.entrypoint"] = override
	mergeSpecials["services.*.env_file"] = mergeToSequence
	mergeSpecials["services.*.environment"] = mergeToSequence
	mergeSpecials["services.*.extra_hosts"] = mergeToSequence
	mergeSpecials["services.*.healthcheck.test"] = override
	mergeSpecials["services.*.labels"] = mergeToSequence
	mergeSpecials["services.*.logging"] = mergeLogging
	mergeSpecials["services.*.networks"] = mergeNetworks
	mergeSpecials["services.*.sysctls"] = mergeToSequence
	mergeSpecials["services.*.tmpfs"] = mergeToSequence
	mergeSpecials["services.*.ulimits.*"] = mergeUlimit
}

// mergeYaml merges map[string]any yaml trees handling special rules
func mergeYaml(e any, o any, p tree.Path) (any, error) {
	for pattern, merger := range mergeSpecials {
		if p.Matches(pattern) {
			merged, err := merger(e, o, p)
			if err != nil {
				return nil, err
			}
			return merged, nil
		}
	}
	if o == nil {
		return e, nil
	}
	switch value := e.(type) {
	case map[string]any:
		other, ok := o.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot override %s", p)
		}
		return mergeMappings(value, other, p)
	case []any:
		other, ok := o.([]any)
		if !ok {
			return nil, fmt.Errorf("cannot override %s", p)
		}
		return append(value, other...), nil
	default:
		return o, nil
	}
}

func mergeMappings(mapping map[string]any, other map[string]any, p tree.Path) (map[string]any, error) {
	for k, v := range other {
		e, ok := mapping[k]
		if !ok || strings.HasPrefix(k, "x-") {
			mapping[k] = v
			continue
		}
		next := p.Next(k)
		merged, err := mergeYaml(e, v, next)
		if err != nil {
			return nil, err
		}
		mapping[k] = merged
	}
	return mapping, nil
}

// logging driver options are merged only when both compose file define the same driver
func mergeLogging(c any, o any, p tree.Path) (any, error) {
	config := c.(map[string]any)
	other := o.(map[string]any)
	// we override logging config if source and override have the same driver set, or none
	d, ok1 := other["driver"]
	o, ok2 := config["driver"]
	if d == o || !ok1 || !ok2 {
		return mergeMappings(config, other, p)
	}
	return other, nil
}

func mergeBuild(c any, o any, path tree.Path) (any, error) {
	toBuild := func(c any) map[string]any {
		switch v := c.(type) {
		case string:
			return map[string]any{
				"context": v,
			}
		case map[string]any:
			return v
		}
		return nil
	}
	return mergeMappings(toBuild(c), toBuild(o), path)
}

func mergeDependsOn(c any, o any, path tree.Path) (any, error) {
	right := convertIntoMapping(c, map[string]any{
		"condition": "service_started",
		"required":  true,
	})
	left := convertIntoMapping(o, map[string]any{
		"condition": "service_started",
		"required":  true,
	})
	return mergeMappings(right, left, path)
}

func mergeNetworks(c any, o any, path tree.Path) (any, error) {
	right := convertIntoMapping(c, nil)
	left := convertIntoMapping(o, nil)
	return mergeMappings(right, left, path)
}

func mergeToSequence(c any, o any, _ tree.Path) (any, error) {
	right := convertIntoSequence(c)
	left := convertIntoSequence(o)
	return append(right, left...), nil
}

func convertIntoSequence(value any) []any {
	switch v := value.(type) {
	case map[string]any:
		seq := make([]any, len(v))
		i := 0
		for k, v := range v {
			if v == nil {
				seq[i] = k
			} else {
				seq[i] = fmt.Sprintf("%s=%v", k, v)
			}
			i++
		}
		slices.SortFunc(seq, func(a, b any) int {
			return cmp.Compare(a.(string), b.(string))
		})
		return seq
	case []any:
		return v
	case string:
		return []any{v}
	}
	return nil
}

func mergeUlimit(_ any, o any, p tree.Path) (any, error) {
	over, ismapping := o.(map[string]any)
	if base, ok := o.(map[string]any); ok && ismapping {
		return mergeMappings(base, over, p)
	}
	return o, nil
}

func mergeIPAMConfig(c any, o any, path tree.Path) (any, error) {
	var ipamConfigs []any
	for _, original := range c.([]any) {
		right := convertIntoMapping(original, nil)
		for _, override := range o.([]any) {
			left := convertIntoMapping(override, nil)
			if left["subnet"] != right["subnet"] {
				// check if left is already in ipamConfigs, add it if not and continue with the next config
				if !slices.ContainsFunc(ipamConfigs, func(a any) bool {
					return a.(map[string]any)["subnet"] == left["subnet"]
				}) {
					ipamConfigs = append(ipamConfigs, left)
					continue
				}
			}
			merged, err := mergeMappings(right, left, path)
			if err != nil {
				return nil, err
			}
			// find index of potential previous config with the same subnet in ipamConfigs
			indexIfExist := slices.IndexFunc(ipamConfigs, func(a any) bool {
				return a.(map[string]any)["subnet"] == merged["subnet"]
			})
			// if a previous config is already in ipamConfigs, replace it
			if indexIfExist >= 0 {
				ipamConfigs[indexIfExist] = merged
			} else {
				// or add the new config to ipamConfigs
				ipamConfigs = append(ipamConfigs, merged)
			}
		}
	}
	return ipamConfigs, nil
}

func convertIntoMapping(a any, defaultValue map[string]any) map[string]any {
	switch v := a.(type) {
	case map[string]any:
		return v
	case []any:
		converted := map[string]any{}
		for _, s := range v {
			if defaultValue == nil {
				converted[s.(string)] = nil
			} else {
				// Create a new map for each key
				converted[s.(string)] = copyMap(defaultValue)
			}
		}
		return converted
	}
	return nil
}

func copyMap(m map[string]any) map[string]any {
	c := make(map[string]any)
	for k, v := range m {
		c[k] = v
	}
	return c
}

func override(_ any, other any, _ tree.Path) (any, error) {
	return other, nil
}
