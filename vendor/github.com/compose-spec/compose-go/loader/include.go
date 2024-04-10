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
	"context"
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/compose-spec/compose-go/dotenv"
	interp "github.com/compose-spec/compose-go/interpolation"
	"github.com/compose-spec/compose-go/types"
	"github.com/pkg/errors"
)

// LoadIncludeConfig parse the require config from raw yaml
func LoadIncludeConfig(source []interface{}) ([]types.IncludeConfig, error) {
	var requires []types.IncludeConfig
	err := Transform(source, &requires)
	return requires, err
}

var transformIncludeConfig TransformerFunc = func(data interface{}) (interface{}, error) {
	switch value := data.(type) {
	case string:
		return map[string]interface{}{"path": value}, nil
	case map[string]interface{}:
		return value, nil
	default:
		return data, errors.Errorf("invalid type %T for `include` configuration", value)
	}
}

func loadInclude(ctx context.Context, filename string, configDetails types.ConfigDetails, model *types.Config, options *Options, loaded []string) (*types.Config, map[string][]types.IncludeConfig, error) {
	included := make(map[string][]types.IncludeConfig)
	for _, r := range model.Include {
		included[filename] = append(included[filename], r)

		for i, p := range r.Path {
			for _, loader := range options.ResourceLoaders {
				if loader.Accept(p) {
					path, err := loader.Load(ctx, p)
					if err != nil {
						return nil, nil, err
					}
					p = path
					break
				}
			}
			r.Path[i] = absPath(configDetails.WorkingDir, p)
		}
		if r.ProjectDirectory == "" {
			r.ProjectDirectory = filepath.Dir(r.Path[0])
		}

		loadOptions := options.clone()
		loadOptions.SetProjectName(model.Name, true)
		loadOptions.ResolvePaths = true
		loadOptions.SkipNormalization = true
		loadOptions.SkipConsistencyCheck = true

		envFromFile, err := dotenv.GetEnvFromFile(configDetails.Environment, r.ProjectDirectory, r.EnvFile)
		if err != nil {
			return nil, nil, err
		}

		config := types.ConfigDetails{
			WorkingDir:  r.ProjectDirectory,
			ConfigFiles: types.ToConfigFiles(r.Path),
			Environment: configDetails.Environment.Clone().Merge(envFromFile),
		}
		loadOptions.Interpolate = &interp.Options{
			Substitute:      options.Interpolate.Substitute,
			LookupValue:     config.LookupEnv,
			TypeCastMapping: options.Interpolate.TypeCastMapping,
		}
		imported, err := load(ctx, config, loadOptions, loaded)
		if err != nil {
			return nil, nil, err
		}
		for k, v := range imported.IncludeReferences {
			included[k] = append(included[k], v...)
		}

		err = importResources(model, imported, r.Path)
		if err != nil {
			return nil, nil, err
		}
	}
	model.Include = nil
	return model, included, nil
}

// importResources import into model all resources defined by imported, and report error on conflict
func importResources(model *types.Config, imported *types.Project, path []string) error {
	services := mapByName(model.Services)
	for _, service := range imported.Services {
		if present, ok := services[service.Name]; ok {
			if reflect.DeepEqual(present, service) {
				continue
			}
			return fmt.Errorf("imported compose file %s defines conflicting service %s", path, service.Name)
		}
		model.Services = append(model.Services, service)
	}
	for _, service := range imported.DisabledServices {
		if disabled, ok := services[service.Name]; ok {
			if reflect.DeepEqual(disabled, service) {
				continue
			}
			return fmt.Errorf("imported compose file %s defines conflicting service %s", path, service.Name)
		}
		model.Services = append(model.Services, service)
	}
	for n, network := range imported.Networks {
		if present, ok := model.Networks[n]; ok {
			if reflect.DeepEqual(present, network) {
				continue
			}
			return fmt.Errorf("imported compose file %s defines conflicting network %s", path, n)
		}
		model.Networks[n] = network
	}
	for n, volume := range imported.Volumes {
		if present, ok := model.Volumes[n]; ok {
			if reflect.DeepEqual(present, volume) {
				continue
			}
			return fmt.Errorf("imported compose file %s defines conflicting volume %s", path, n)
		}
		model.Volumes[n] = volume
	}
	for n, secret := range imported.Secrets {
		if present, ok := model.Secrets[n]; ok {
			if reflect.DeepEqual(present, secret) {
				continue
			}
			return fmt.Errorf("imported compose file %s defines conflicting secret %s", path, n)
		}
		model.Secrets[n] = secret
	}
	for n, config := range imported.Configs {
		if present, ok := model.Configs[n]; ok {
			if reflect.DeepEqual(present, config) {
				continue
			}
			return fmt.Errorf("imported compose file %s defines conflicting config %s", path, n)
		}
		model.Configs[n] = config
	}
	return nil
}
