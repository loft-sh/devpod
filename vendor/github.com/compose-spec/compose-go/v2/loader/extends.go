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

	"github.com/compose-spec/compose-go/v2/consts"
	"github.com/compose-spec/compose-go/v2/override"
	"github.com/compose-spec/compose-go/v2/paths"
	"github.com/compose-spec/compose-go/v2/types"
)

func ApplyExtends(ctx context.Context, dict map[string]any, opts *Options, tracker *cycleTracker, post ...PostProcessor) error {
	a, ok := dict["services"]
	if !ok {
		return nil
	}
	services, ok := a.(map[string]any)
	if !ok {
		return fmt.Errorf("services must be a mapping")
	}
	for name := range services {
		merged, err := applyServiceExtends(ctx, name, services, opts, tracker, post...)
		if err != nil {
			return err
		}
		services[name] = merged
	}
	dict["services"] = services
	return nil
}

func applyServiceExtends(ctx context.Context, name string, services map[string]any, opts *Options, tracker *cycleTracker, post ...PostProcessor) (any, error) {
	s := services[name]
	if s == nil {
		return nil, nil
	}
	service, ok := s.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("services.%s must be a mapping", name)
	}
	extends, ok := service["extends"]
	if !ok {
		return s, nil
	}
	filename := ctx.Value(consts.ComposeFileKey{}).(string)
	var (
		err  error
		ref  string
		file any
	)
	switch v := extends.(type) {
	case map[string]any:
		ref = v["service"].(string)
		file = v["file"]
		opts.ProcessEvent("extends", v)
	case string:
		ref = v
		opts.ProcessEvent("extends", map[string]any{"service": ref})
	}

	var (
		base      any
		processor PostProcessor
	)

	if file != nil {
		refFilename := file.(string)
		services, processor, err = getExtendsBaseFromFile(ctx, name, ref, filename, refFilename, opts, tracker)
		post = append(post, processor)
		if err != nil {
			return nil, err
		}
		filename = refFilename
	} else {
		_, ok := services[ref]
		if !ok {
			return nil, fmt.Errorf("cannot extend service %q in %s: service %q not found", name, filename, ref)
		}
	}

	tracker, err = tracker.Add(filename, name)
	if err != nil {
		return nil, err
	}

	// recursively apply `extends`
	base, err = applyServiceExtends(ctx, ref, services, opts, tracker, post...)
	if err != nil {
		return nil, err
	}

	if base == nil {
		return service, nil
	}
	source := deepClone(base).(map[string]any)

	for _, processor := range post {
		processor.Apply(map[string]any{
			"services": map[string]any{
				name: source,
			},
		})
	}
	merged, err := override.ExtendService(source, service)
	if err != nil {
		return nil, err
	}
	delete(merged, "extends")
	services[name] = merged
	return merged, nil
}

func getExtendsBaseFromFile(
	ctx context.Context,
	name, ref string,
	path, refPath string,
	opts *Options,
	ct *cycleTracker,
) (map[string]any, PostProcessor, error) {
	for _, loader := range opts.ResourceLoaders {
		if !loader.Accept(refPath) {
			continue
		}
		local, err := loader.Load(ctx, refPath)
		if err != nil {
			return nil, nil, err
		}
		localdir := filepath.Dir(local)
		relworkingdir := loader.Dir(refPath)

		extendsOpts := opts.clone()
		// replace localResourceLoader with a new flavour, using extended file base path
		extendsOpts.ResourceLoaders = append(opts.RemoteResourceLoaders(), localResourceLoader{
			WorkingDir: localdir,
		})
		extendsOpts.ResolvePaths = false // we do relative path resolution after file has been loaded
		extendsOpts.SkipNormalization = true
		extendsOpts.SkipConsistencyCheck = true
		extendsOpts.SkipInclude = true
		extendsOpts.SkipExtends = true    // we manage extends recursively based on raw service definition
		extendsOpts.SkipValidation = true // we validate the merge result
		extendsOpts.SkipDefaultValues = true
		source, processor, err := loadYamlFile(ctx, types.ConfigFile{Filename: local},
			extendsOpts, relworkingdir, nil, ct, map[string]any{}, nil)
		if err != nil {
			return nil, nil, err
		}
		m, ok := source["services"]
		if !ok {
			return nil, nil, fmt.Errorf("cannot extend service %q in %s: no services section", name, local)
		}
		services, ok := m.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("cannot extend service %q in %s: services must be a mapping", name, local)
		}
		_, ok = services[ref]
		if !ok {
			return nil, nil, fmt.Errorf(
				"cannot extend service %q in %s: service %q not found in %s",
				name,
				path,
				ref,
				refPath,
			)
		}

		var remotes []paths.RemoteResource
		for _, loader := range opts.RemoteResourceLoaders() {
			remotes = append(remotes, loader.Accept)
		}
		err = paths.ResolveRelativePaths(source, relworkingdir, remotes)
		if err != nil {
			return nil, nil, err
		}

		return services, processor, nil
	}
	return nil, nil, fmt.Errorf("cannot read %s", refPath)
}

func deepClone(value any) any {
	switch v := value.(type) {
	case []any:
		cp := make([]any, len(v))
		for i, e := range v {
			cp[i] = deepClone(e)
		}
		return cp
	case map[string]any:
		cp := make(map[string]any, len(v))
		for k, e := range v {
			cp[k] = deepClone(e)
		}
		return cp
	default:
		return value
	}
}
