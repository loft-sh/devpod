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

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
)

// Normalize compose project by moving deprecated attributes to their canonical position and injecting implicit defaults
func Normalize(dict map[string]any, env types.Mapping) (map[string]any, error) {
	normalizeNetworks(dict)

	if d, ok := dict["services"]; ok {
		services := d.(map[string]any)
		for name, s := range services {
			service := s.(map[string]any)

			if service["pull_policy"] == types.PullPolicyIfNotPresent {
				service["pull_policy"] = types.PullPolicyMissing
			}

			fn := func(s string) (string, bool) {
				v, ok := env[s]
				return v, ok
			}

			if b, ok := service["build"]; ok {
				build := b.(map[string]any)
				if build["context"] == nil {
					build["context"] = "."
				}
				if build["dockerfile"] == nil && build["dockerfile_inline"] == nil {
					build["dockerfile"] = "Dockerfile"
				}

				if a, ok := build["args"]; ok {
					build["args"], _ = resolve(a, fn, false)
				}

				service["build"] = build
			}

			if e, ok := service["environment"]; ok {
				service["environment"], _ = resolve(e, fn, true)
			}

			var dependsOn map[string]any
			if d, ok := service["depends_on"]; ok {
				dependsOn = d.(map[string]any)
			} else {
				dependsOn = map[string]any{}
			}
			if l, ok := service["links"]; ok {
				links := l.([]any)
				for _, e := range links {
					link := e.(string)
					parts := strings.Split(link, ":")
					if len(parts) == 2 {
						link = parts[0]
					}
					if _, ok := dependsOn[link]; !ok {
						dependsOn[link] = map[string]any{
							"condition": types.ServiceConditionStarted,
							"restart":   true,
							"required":  true,
						}
					}
				}
			}

			for _, namespace := range []string{"network_mode", "ipc", "pid", "uts", "cgroup"} {
				if n, ok := service[namespace]; ok {
					ref := n.(string)
					if strings.HasPrefix(ref, types.ServicePrefix) {
						shared := ref[len(types.ServicePrefix):]
						if _, ok := dependsOn[shared]; !ok {
							dependsOn[shared] = map[string]any{
								"condition": types.ServiceConditionStarted,
								"restart":   true,
								"required":  true,
							}
						}
					}
				}
			}

			if n, ok := service["volumes_from"]; ok {
				volumesFrom := n.([]any)
				for _, v := range volumesFrom {
					vol := v.(string)
					if !strings.HasPrefix(vol, types.ContainerPrefix) {
						spec := strings.Split(vol, ":")
						if _, ok := dependsOn[spec[0]]; !ok {
							dependsOn[spec[0]] = map[string]any{
								"condition": types.ServiceConditionStarted,
								"restart":   false,
								"required":  true,
							}
						}
					}
				}
			}
			if len(dependsOn) > 0 {
				service["depends_on"] = dependsOn
			}
			services[name] = service
		}
		dict["services"] = services
	}

	setNameFromKey(dict)

	return dict, nil
}

func normalizeNetworks(dict map[string]any) {
	var networks map[string]any
	if n, ok := dict["networks"]; ok {
		networks = n.(map[string]any)
	} else {
		networks = map[string]any{}
	}

	// implicit `default` network must be introduced only if actually used by some service
	usesDefaultNetwork := false

	if s, ok := dict["services"]; ok {
		services := s.(map[string]any)
		for name, se := range services {
			service := se.(map[string]any)
			if _, ok := service["network_mode"]; ok {
				continue
			}
			if n, ok := service["networks"]; !ok {
				// If none explicitly declared, service is connected to default network
				service["networks"] = map[string]any{"default": nil}
				usesDefaultNetwork = true
			} else {
				net := n.(map[string]any)
				if len(net) == 0 {
					// networks section declared but empty (corner case)
					service["networks"] = map[string]any{"default": nil}
					usesDefaultNetwork = true
				} else if _, ok := net["default"]; ok {
					usesDefaultNetwork = true
				}
			}
			services[name] = service
		}
		dict["services"] = services
	}

	if _, ok := networks["default"]; !ok && usesDefaultNetwork {
		// If not declared explicitly, Compose model involves an implicit "default" network
		networks["default"] = nil
	}

	if len(networks) > 0 {
		dict["networks"] = networks
	}
}

func resolve(a any, fn func(s string) (string, bool), keepEmpty bool) (any, bool) {
	switch v := a.(type) {
	case []any:
		var resolved []any
		for _, val := range v {
			if r, ok := resolve(val, fn, keepEmpty); ok {
				resolved = append(resolved, r)
			}
		}
		return resolved, true
	case map[string]any:
		resolved := map[string]any{}
		for key, val := range v {
			if val != nil {
				resolved[key] = val
				continue
			}
			if s, ok := fn(key); ok {
				resolved[key] = s
			} else if keepEmpty {
				resolved[key] = nil
			}
		}
		return resolved, true
	case string:
		if !strings.Contains(v, "=") {
			if val, ok := fn(v); ok {
				return fmt.Sprintf("%s=%s", v, val), true
			}
			if keepEmpty {
				return v, true
			}
			return "", false
		}
		return v, true
	default:
		return v, false
	}
}

// Resources with no explicit name are actually named by their key in map
func setNameFromKey(dict map[string]any) {
	for _, r := range []string{"networks", "volumes", "configs", "secrets"} {
		a, ok := dict[r]
		if !ok {
			continue
		}
		toplevel := a.(map[string]any)
		for key, r := range toplevel {
			var resource map[string]any
			if r != nil {
				resource = r.(map[string]any)
			} else {
				resource = map[string]any{}
			}
			if resource["name"] == nil {
				if x, ok := resource["external"]; ok && isTrue(x) {
					resource["name"] = key
				} else {
					resource["name"] = fmt.Sprintf("%s_%s", dict["name"], key)
				}
			}
			toplevel[key] = resource
		}
	}
}

func isTrue(x any) bool {
	parseBool, _ := strconv.ParseBool(fmt.Sprint(x))
	return parseBool
}
