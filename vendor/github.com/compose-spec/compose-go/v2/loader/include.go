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
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/compose-spec/compose-go/v2/dotenv"
	interp "github.com/compose-spec/compose-go/v2/interpolation"
	"github.com/compose-spec/compose-go/v2/types"
)

// loadIncludeConfig parse the required config from raw yaml
func loadIncludeConfig(source any) ([]types.IncludeConfig, error) {
	if source == nil {
		return nil, nil
	}
	configs, ok := source.([]any)
	if !ok {
		return nil, fmt.Errorf("`include` must be a list, got %s", source)
	}
	for i, config := range configs {
		if v, ok := config.(string); ok {
			configs[i] = map[string]any{
				"path": v,
			}
		}
	}
	var requires []types.IncludeConfig
	err := Transform(source, &requires)
	return requires, err
}

func ApplyInclude(ctx context.Context, workingDir string, environment types.Mapping, model map[string]any, options *Options, included []string) error {
	includeConfig, err := loadIncludeConfig(model["include"])
	if err != nil {
		return err
	}

	for _, r := range includeConfig {
		for _, listener := range options.Listeners {
			listener("include", map[string]any{
				"path":       r.Path,
				"workingdir": workingDir,
			})
		}

		var relworkingdir string
		for i, p := range r.Path {
			for _, loader := range options.ResourceLoaders {
				if !loader.Accept(p) {
					continue
				}
				path, err := loader.Load(ctx, p)
				if err != nil {
					return err
				}
				p = path

				if i == 0 { // This is the "main" file, used to define project-directory. Others are overrides

					switch {
					case r.ProjectDirectory == "":
						relworkingdir = loader.Dir(path)
						r.ProjectDirectory = filepath.Dir(path)
					case !filepath.IsAbs(r.ProjectDirectory):
						relworkingdir = loader.Dir(r.ProjectDirectory)
						r.ProjectDirectory = filepath.Join(workingDir, r.ProjectDirectory)

					default:
						relworkingdir = r.ProjectDirectory

					}
					for _, f := range included {
						if f == path {
							included = append(included, path)
							return fmt.Errorf("include cycle detected:\n%s\n include %s", included[0], strings.Join(included[1:], "\n include "))
						}
					}
				}
			}
			r.Path[i] = p
		}

		loadOptions := options.clone()
		loadOptions.ResolvePaths = true
		loadOptions.SkipNormalization = true
		loadOptions.SkipConsistencyCheck = true
		loadOptions.ResourceLoaders = append(loadOptions.RemoteResourceLoaders(), localResourceLoader{
			WorkingDir: r.ProjectDirectory,
		})

		if len(r.EnvFile) == 0 {
			f := filepath.Join(r.ProjectDirectory, ".env")
			if s, err := os.Stat(f); err == nil && !s.IsDir() {
				r.EnvFile = types.StringList{f}
			}
		} else {
			envFile := []string{}
			for _, f := range r.EnvFile {
				if !filepath.IsAbs(f) {
					f = filepath.Join(workingDir, f)
					s, err := os.Stat(f)
					if err != nil {
						return err
					}
					if s.IsDir() {
						return fmt.Errorf("%s is not a file", f)
					}
				}
				envFile = append(envFile, f)
			}
			r.EnvFile = envFile
		}

		envFromFile, err := dotenv.GetEnvFromFile(environment, r.EnvFile)
		if err != nil {
			return err
		}

		config := types.ConfigDetails{
			WorkingDir:  relworkingdir,
			ConfigFiles: types.ToConfigFiles(r.Path),
			Environment: environment.Clone().Merge(envFromFile),
		}
		loadOptions.Interpolate = &interp.Options{
			Substitute:      options.Interpolate.Substitute,
			LookupValue:     config.LookupEnv,
			TypeCastMapping: options.Interpolate.TypeCastMapping,
		}
		imported, err := loadYamlModel(ctx, config, loadOptions, &cycleTracker{}, included)
		if err != nil {
			return err
		}
		err = importResources(imported, model)
		if err != nil {
			return err
		}
	}
	delete(model, "include")
	return nil
}

// importResources import into model all resources defined by imported, and report error on conflict
func importResources(source map[string]any, target map[string]any) error {
	if err := importResource(source, target, "services"); err != nil {
		return err
	}
	if err := importResource(source, target, "volumes"); err != nil {
		return err
	}
	if err := importResource(source, target, "networks"); err != nil {
		return err
	}
	if err := importResource(source, target, "secrets"); err != nil {
		return err
	}
	if err := importResource(source, target, "configs"); err != nil {
		return err
	}
	return nil
}

func importResource(source map[string]any, target map[string]any, key string) error {
	from := source[key]
	if from != nil {
		var to map[string]any
		if v, ok := target[key]; ok {
			to = v.(map[string]any)
		} else {
			to = map[string]any{}
		}
		for name, a := range from.(map[string]any) {
			if conflict, ok := to[name]; ok {
				if reflect.DeepEqual(a, conflict) {
					continue
				}
				return fmt.Errorf("%s.%s conflicts with imported resource", key, name)
			}
			to[name] = a
		}
		target[key] = to
	}
	return nil
}
