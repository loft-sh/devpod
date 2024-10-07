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

	"github.com/compose-spec/compose-go/v2/types"
)

// ResolveEnvironment update the environment variables for the format {- VAR} (without interpolation)
func ResolveEnvironment(dict map[string]any, environment types.Mapping) {
	resolveServicesEnvironment(dict, environment)
	resolveSecretsEnvironment(dict, environment)
	resolveConfigsEnvironment(dict, environment)
}

func resolveServicesEnvironment(dict map[string]any, environment types.Mapping) {
	services, ok := dict["services"].(map[string]any)
	if !ok {
		return
	}

	for service, cfg := range services {
		serviceConfig, ok := cfg.(map[string]any)
		if !ok {
			continue
		}
		serviceEnv, ok := serviceConfig["environment"].([]any)
		if !ok {
			continue
		}
		envs := []any{}
		for _, env := range serviceEnv {
			varEnv, ok := env.(string)
			if !ok {
				continue
			}
			if found, ok := environment[varEnv]; ok {
				envs = append(envs, fmt.Sprintf("%s=%s", varEnv, found))
			} else {
				// either does not exist or it was already resolved in interpolation
				envs = append(envs, varEnv)
			}
		}
		serviceConfig["environment"] = envs
		services[service] = serviceConfig
	}
	dict["services"] = services
}

func resolveSecretsEnvironment(dict map[string]any, environment types.Mapping) {
	secrets, ok := dict["secrets"].(map[string]any)
	if !ok {
		return
	}

	for name, cfg := range secrets {
		secret, ok := cfg.(map[string]any)
		if !ok {
			continue
		}
		env, ok := secret["environment"].(string)
		if !ok {
			continue
		}
		if found, ok := environment[env]; ok {
			secret[types.SecretConfigXValue] = found
		}
		secrets[name] = secret
	}
	dict["secrets"] = secrets
}

func resolveConfigsEnvironment(dict map[string]any, environment types.Mapping) {
	configs, ok := dict["configs"].(map[string]any)
	if !ok {
		return
	}

	for name, cfg := range configs {
		config, ok := cfg.(map[string]any)
		if !ok {
			continue
		}
		env, ok := config["environment"].(string)
		if !ok {
			continue
		}
		if found, ok := environment[env]; ok {
			config["content"] = found
		}
		configs[name] = config
	}
	dict["configs"] = configs
}
