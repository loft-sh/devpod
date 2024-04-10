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
	"os"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/types"
)

// ResolveRelativePaths resolves relative paths based on project WorkingDirectory
func ResolveRelativePaths(project *types.Project) error {
	absWorkingDir, err := filepath.Abs(project.WorkingDir)
	if err != nil {
		return err
	}
	project.WorkingDir = absWorkingDir

	absComposeFiles, err := absComposeFiles(project.ComposeFiles)
	if err != nil {
		return err
	}
	project.ComposeFiles = absComposeFiles

	for i, s := range project.Services {
		ResolveServiceRelativePaths(project.WorkingDir, &s)
		project.Services[i] = s
	}

	for i, obj := range project.Configs {
		if obj.File != "" {
			obj.File = absPath(project.WorkingDir, obj.File)
			project.Configs[i] = obj
		}
	}

	for i, obj := range project.Secrets {
		if obj.File != "" {
			obj.File = resolveMaybeUnixPath(project.WorkingDir, obj.File)
			project.Secrets[i] = obj
		}
	}

	for name, config := range project.Volumes {
		if config.Driver == "local" && config.DriverOpts["o"] == "bind" {
			// This is actually a bind mount
			config.DriverOpts["device"] = resolveMaybeUnixPath(project.WorkingDir, config.DriverOpts["device"])
			project.Volumes[name] = config
		}
	}

	// don't coerce a nil map to an empty map
	if project.IncludeReferences != nil {
		absIncludes := make(map[string][]types.IncludeConfig, len(project.IncludeReferences))
		for filename, config := range project.IncludeReferences {
			filename = absPath(project.WorkingDir, filename)
			absConfigs := make([]types.IncludeConfig, len(config))
			for i, c := range config {
				absConfigs[i] = types.IncludeConfig{
					Path:             resolvePaths(project.WorkingDir, c.Path),
					ProjectDirectory: absPath(project.WorkingDir, c.ProjectDirectory),
					EnvFile:          resolvePaths(project.WorkingDir, c.EnvFile),
				}
			}
			absIncludes[filename] = absConfigs
		}
		project.IncludeReferences = absIncludes
	}

	return nil
}

func ResolveServiceRelativePaths(workingDir string, s *types.ServiceConfig) {
	if s.Build != nil {
		if !isRemoteContext(s.Build.Context) {
			s.Build.Context = absPath(workingDir, s.Build.Context)
		}
		for name, path := range s.Build.AdditionalContexts {
			if strings.Contains(path, "://") { // `docker-image://` or any builder specific context type
				continue
			}
			if isRemoteContext(path) {
				continue
			}
			s.Build.AdditionalContexts[name] = absPath(workingDir, path)
		}
	}
	for j, f := range s.EnvFile {
		s.EnvFile[j] = absPath(workingDir, f)
	}

	if s.Extends != nil && s.Extends.File != "" {
		s.Extends.File = absPath(workingDir, s.Extends.File)
	}

	for i, vol := range s.Volumes {
		if vol.Type != types.VolumeTypeBind {
			continue
		}
		s.Volumes[i].Source = resolveMaybeUnixPath(workingDir, vol.Source)
	}

	if s.Develop != nil {
		for i, w := range s.Develop.Watch {
			w.Path = absPath(workingDir, w.Path)
			s.Develop.Watch[i] = w
		}
	}
}

func absPath(workingDir string, filePath string) string {
	if strings.HasPrefix(filePath, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, filePath[1:])
	}
	if filepath.IsAbs(filePath) {
		return filePath
	}
	return filepath.Join(workingDir, filePath)
}

func absComposeFiles(composeFiles []string) ([]string, error) {
	for i, composeFile := range composeFiles {
		absComposefile, err := filepath.Abs(composeFile)
		if err != nil {
			return nil, err
		}
		composeFiles[i] = absComposefile
	}
	return composeFiles, nil
}

// isRemoteContext returns true if the value is a Git reference or HTTP(S) URL.
//
// Any other value is assumed to be a local filesystem path and returns false.
//
// See: https://github.com/moby/buildkit/blob/18fc875d9bfd6e065cd8211abc639434ba65aa56/frontend/dockerui/context.go#L76-L79
func isRemoteContext(maybeURL string) bool {
	for _, prefix := range []string{"https://", "http://", "git://", "ssh://", "github.com/", "git@"} {
		if strings.HasPrefix(maybeURL, prefix) {
			return true
		}
	}
	return false
}

func resolvePaths(basePath string, in types.StringList) types.StringList {
	if in == nil {
		return nil
	}
	ret := make(types.StringList, len(in))
	for i := range in {
		ret[i] = absPath(basePath, in[i])
	}
	return ret
}
