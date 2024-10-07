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

package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/compose-spec/compose-go/v2/consts"
	"github.com/compose-spec/compose-go/v2/dotenv"
	"github.com/compose-spec/compose-go/v2/errdefs"
	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/compose-spec/compose-go/v2/utils"
)

// ProjectOptions provides common configuration for loading a project.
type ProjectOptions struct {
	// Name is a valid Compose project name to be used or empty.
	//
	// If empty, the project loader will automatically infer a reasonable
	// project name if possible.
	Name string

	// WorkingDir is a file path to use as the project directory or empty.
	//
	// If empty, the project loader will automatically infer a reasonable
	// working directory if possible.
	WorkingDir string

	// ConfigPaths are file paths to one or more Compose files.
	//
	// These are applied in order by the loader following the override logic
	// as described in the spec.
	//
	// The first entry is required and is the primary Compose file.
	// For convenience, WithConfigFileEnv and WithDefaultConfigPath
	// are provided to populate this in a predictable manner.
	ConfigPaths []string

	// Environment are additional environment variables to make available
	// for interpolation.
	//
	// NOTE: For security, the loader does not automatically expose any
	// process environment variables. For convenience, WithOsEnv can be
	// used if appropriate.
	Environment types.Mapping

	// EnvFiles are file paths to ".env" files with additional environment
	// variable data.
	//
	// These are loaded in-order, so it is possible to override variables or
	// in subsequent files.
	//
	// This field is optional, but any file paths that are included here must
	// exist or an error will be returned during load.
	EnvFiles []string

	loadOptions []func(*loader.Options)

	// Callbacks to retrieve metadata information during parse defined before
	// creating the project
	Listeners []loader.Listener
}

type ProjectOptionsFn func(*ProjectOptions) error

// NewProjectOptions creates ProjectOptions
func NewProjectOptions(configs []string, opts ...ProjectOptionsFn) (*ProjectOptions, error) {
	options := &ProjectOptions{
		ConfigPaths: configs,
		Environment: map[string]string{},
		Listeners:   []loader.Listener{},
	}
	for _, o := range opts {
		err := o(options)
		if err != nil {
			return nil, err
		}
	}
	return options, nil
}

// WithName defines ProjectOptions' name
func WithName(name string) ProjectOptionsFn {
	return func(o *ProjectOptions) error {
		// a project (once loaded) cannot have an empty name
		// however, on the options object, the name is optional: if unset,
		// a name will be inferred by the loader, so it's legal to set the
		// name to an empty string here
		if name != loader.NormalizeProjectName(name) {
			return loader.InvalidProjectNameErr(name)
		}
		o.Name = name
		return nil
	}
}

// WithWorkingDirectory defines ProjectOptions' working directory
func WithWorkingDirectory(wd string) ProjectOptionsFn {
	return func(o *ProjectOptions) error {
		if wd == "" {
			return nil
		}
		abs, err := filepath.Abs(wd)
		if err != nil {
			return err
		}
		o.WorkingDir = abs
		return nil
	}
}

// WithConfigFileEnv allow to set compose config file paths by COMPOSE_FILE environment variable
func WithConfigFileEnv(o *ProjectOptions) error {
	if len(o.ConfigPaths) > 0 {
		return nil
	}
	sep := o.Environment[consts.ComposePathSeparator]
	if sep == "" {
		sep = string(os.PathListSeparator)
	}
	f, ok := o.Environment[consts.ComposeFilePath]
	if ok {
		paths, err := absolutePaths(strings.Split(f, sep))
		o.ConfigPaths = paths
		return err
	}
	return nil
}

// WithDefaultConfigPath searches for default config files from working directory
func WithDefaultConfigPath(o *ProjectOptions) error {
	if len(o.ConfigPaths) > 0 {
		return nil
	}
	pwd, err := o.GetWorkingDir()
	if err != nil {
		return err
	}
	for {
		candidates := findFiles(DefaultFileNames, pwd)
		if len(candidates) > 0 {
			winner := candidates[0]
			if len(candidates) > 1 {
				logrus.Warnf("Found multiple config files with supported names: %s", strings.Join(candidates, ", "))
				logrus.Warnf("Using %s", winner)
			}
			o.ConfigPaths = append(o.ConfigPaths, winner)

			overrides := findFiles(DefaultOverrideFileNames, pwd)
			if len(overrides) > 0 {
				if len(overrides) > 1 {
					logrus.Warnf("Found multiple override files with supported names: %s", strings.Join(overrides, ", "))
					logrus.Warnf("Using %s", overrides[0])
				}
				o.ConfigPaths = append(o.ConfigPaths, overrides[0])
			}
			return nil
		}
		parent := filepath.Dir(pwd)
		if parent == pwd {
			// no config file found, but that's not a blocker if caller only needs project name
			return nil
		}
		pwd = parent
	}
}

// WithEnv defines a key=value set of variables used for compose file interpolation
func WithEnv(env []string) ProjectOptionsFn {
	return func(o *ProjectOptions) error {
		for k, v := range utils.GetAsEqualsMap(env) {
			o.Environment[k] = v
		}
		return nil
	}
}

// WithDiscardEnvFile sets discards the `env_file` section after resolving to
// the `environment` section
func WithDiscardEnvFile(o *ProjectOptions) error {
	o.loadOptions = append(o.loadOptions, loader.WithDiscardEnvFiles)
	return nil
}

// WithLoadOptions provides a hook to control how compose files are loaded
func WithLoadOptions(loadOptions ...func(*loader.Options)) ProjectOptionsFn {
	return func(o *ProjectOptions) error {
		o.loadOptions = append(o.loadOptions, loadOptions...)
		return nil
	}
}

// WithDefaultProfiles uses the provided profiles (if any), and falls back to
// profiles specified via the COMPOSE_PROFILES environment variable otherwise.
func WithDefaultProfiles(profiles ...string) ProjectOptionsFn {
	return func(o *ProjectOptions) error {
		if len(profiles) == 0 {
			for _, s := range strings.Split(o.Environment[consts.ComposeProfiles], ",") {
				profiles = append(profiles, strings.TrimSpace(s))
			}
		}
		o.loadOptions = append(o.loadOptions, loader.WithProfiles(profiles))
		return nil
	}
}

// WithProfiles sets profiles to be activated
func WithProfiles(profiles []string) ProjectOptionsFn {
	return func(o *ProjectOptions) error {
		o.loadOptions = append(o.loadOptions, loader.WithProfiles(profiles))
		return nil
	}
}

// WithOsEnv imports environment variables from OS
func WithOsEnv(o *ProjectOptions) error {
	for k, v := range utils.GetAsEqualsMap(os.Environ()) {
		if _, set := o.Environment[k]; set {
			continue
		}
		o.Environment[k] = v
	}
	return nil
}

// WithEnvFile sets an alternate env file.
//
// Deprecated: use WithEnvFiles instead.
func WithEnvFile(file string) ProjectOptionsFn {
	var files []string
	if file != "" {
		files = []string{file}
	}
	return WithEnvFiles(files...)
}

// WithEnvFiles set env file(s) to be loaded to set project environment.
// defaults to local .env file if no explicit file is selected, until COMPOSE_DISABLE_ENV_FILE is set
func WithEnvFiles(file ...string) ProjectOptionsFn {
	return func(o *ProjectOptions) error {
		if len(file) > 0 {
			o.EnvFiles = file
			return nil
		}
		if v, ok := os.LookupEnv(consts.ComposeDisableDefaultEnvFile); ok {
			b, err := strconv.ParseBool(v)
			if err != nil {
				return err
			}
			if b {
				return nil
			}
		}

		wd, err := o.GetWorkingDir()
		if err != nil {
			return err
		}
		defaultDotEnv := filepath.Join(wd, ".env")

		s, err := os.Stat(defaultDotEnv)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if !s.IsDir() {
			o.EnvFiles = []string{defaultDotEnv}
		}
		return nil
	}
}

// WithDotEnv imports environment variables from .env file
func WithDotEnv(o *ProjectOptions) error {
	envMap, err := dotenv.GetEnvFromFile(o.Environment, o.EnvFiles)
	if err != nil {
		return err
	}
	o.Environment.Merge(envMap)
	return nil
}

// WithInterpolation set ProjectOptions to enable/skip interpolation
func WithInterpolation(interpolation bool) ProjectOptionsFn {
	return func(o *ProjectOptions) error {
		o.loadOptions = append(o.loadOptions, func(options *loader.Options) {
			options.SkipInterpolation = !interpolation
		})
		return nil
	}
}

// WithNormalization set ProjectOptions to enable/skip normalization
func WithNormalization(normalization bool) ProjectOptionsFn {
	return func(o *ProjectOptions) error {
		o.loadOptions = append(o.loadOptions, func(options *loader.Options) {
			options.SkipNormalization = !normalization
		})
		return nil
	}
}

// WithConsistency set ProjectOptions to enable/skip consistency
func WithConsistency(consistency bool) ProjectOptionsFn {
	return func(o *ProjectOptions) error {
		o.loadOptions = append(o.loadOptions, func(options *loader.Options) {
			options.SkipConsistencyCheck = !consistency
		})
		return nil
	}
}

// WithResolvedPaths set ProjectOptions to enable paths resolution
func WithResolvedPaths(resolve bool) ProjectOptionsFn {
	return func(o *ProjectOptions) error {
		o.loadOptions = append(o.loadOptions, func(options *loader.Options) {
			options.ResolvePaths = resolve
		})
		return nil
	}
}

// WithResourceLoader register support for ResourceLoader to manage remote resources
func WithResourceLoader(r loader.ResourceLoader) ProjectOptionsFn {
	return func(o *ProjectOptions) error {
		o.loadOptions = append(o.loadOptions, func(options *loader.Options) {
			options.ResourceLoaders = append(options.ResourceLoaders, r)
		})
		return nil
	}
}

// WithExtension register a know extension `x-*` with the go struct type to decode into
func WithExtension(name string, typ any) ProjectOptionsFn {
	return func(o *ProjectOptions) error {
		o.loadOptions = append(o.loadOptions, func(options *loader.Options) {
			if options.KnownExtensions == nil {
				options.KnownExtensions = map[string]any{}
			}
			options.KnownExtensions[name] = typ
		})
		return nil
	}
}

// Append listener to event
func (o *ProjectOptions) WithListeners(listeners ...loader.Listener) {
	o.Listeners = append(o.Listeners, listeners...)
}

// WithoutEnvironmentResolution disable environment resolution
func WithoutEnvironmentResolution(o *ProjectOptions) error {
	o.loadOptions = append(o.loadOptions, func(options *loader.Options) {
		options.SkipResolveEnvironment = true
	})
	return nil
}

// DefaultFileNames defines the Compose file names for auto-discovery (in order of preference)
var DefaultFileNames = []string{"compose.yaml", "compose.yml", "docker-compose.yml", "docker-compose.yaml"}

// DefaultOverrideFileNames defines the Compose override file names for auto-discovery (in order of preference)
var DefaultOverrideFileNames = []string{"compose.override.yml", "compose.override.yaml", "docker-compose.override.yml", "docker-compose.override.yaml"}

func (o *ProjectOptions) GetWorkingDir() (string, error) {
	if o.WorkingDir != "" {
		return filepath.Abs(o.WorkingDir)
	}
	for _, path := range o.ConfigPaths {
		if path != "-" {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return "", err
			}
			return filepath.Dir(absPath), nil
		}
	}
	return os.Getwd()
}

func (o *ProjectOptions) GetConfigFiles() ([]types.ConfigFile, error) {
	configPaths, err := o.getConfigPaths()
	if err != nil {
		return nil, err
	}

	var configs []types.ConfigFile
	for _, f := range configPaths {
		var b []byte
		if f == "-" {
			b, err = io.ReadAll(os.Stdin)
			if err != nil {
				return nil, err
			}
		} else {
			f, err := filepath.Abs(f)
			if err != nil {
				return nil, err
			}
			b, err = os.ReadFile(f)
			if err != nil {
				return nil, err
			}
		}
		configs = append(configs, types.ConfigFile{
			Filename: f,
			Content:  b,
		})
	}
	return configs, err
}

// LoadProject loads compose file according to options and bind to types.Project go structs
func (o *ProjectOptions) LoadProject(ctx context.Context) (*types.Project, error) {
	configDetails, err := o.prepare()
	if err != nil {
		return nil, err
	}

	project, err := loader.LoadWithContext(ctx, configDetails, o.loadOptions...)
	if err != nil {
		return nil, err
	}

	for _, config := range configDetails.ConfigFiles {
		project.ComposeFiles = append(project.ComposeFiles, config.Filename)
	}

	return project, nil
}

// LoadModel loads compose file according to options and returns a raw (yaml tree) model
func (o *ProjectOptions) LoadModel(ctx context.Context) (map[string]any, error) {
	configDetails, err := o.prepare()
	if err != nil {
		return nil, err
	}

	return loader.LoadModelWithContext(ctx, configDetails, o.loadOptions...)
}

// prepare converts ProjectOptions into loader's types.ConfigDetails and configures default load options
func (o *ProjectOptions) prepare() (types.ConfigDetails, error) {
	configs, err := o.GetConfigFiles()
	if err != nil {
		return types.ConfigDetails{}, err
	}

	workingDir, err := o.GetWorkingDir()
	if err != nil {
		return types.ConfigDetails{}, err
	}

	configDetails := types.ConfigDetails{
		ConfigFiles: configs,
		WorkingDir:  workingDir,
		Environment: o.Environment,
	}

	o.loadOptions = append(o.loadOptions,
		withNamePrecedenceLoad(workingDir, o),
		withConvertWindowsPaths(o),
		withListeners(o))
	return configDetails, nil
}

// ProjectFromOptions load a compose project based on command line options
// Deprecated: use ProjectOptions.LoadProject or ProjectOptions.LoadModel
func ProjectFromOptions(ctx context.Context, options *ProjectOptions) (*types.Project, error) {
	return options.LoadProject(ctx)
}

func withNamePrecedenceLoad(absWorkingDir string, options *ProjectOptions) func(*loader.Options) {
	return func(opts *loader.Options) {
		if options.Name != "" {
			opts.SetProjectName(options.Name, true)
		} else if nameFromEnv, ok := options.Environment[consts.ComposeProjectName]; ok && nameFromEnv != "" {
			opts.SetProjectName(nameFromEnv, true)
		} else {
			opts.SetProjectName(
				loader.NormalizeProjectName(filepath.Base(absWorkingDir)),
				false,
			)
		}
	}
}

func withConvertWindowsPaths(options *ProjectOptions) func(*loader.Options) {
	return func(o *loader.Options) {
		if o.ResolvePaths {
			o.ConvertWindowsPaths = utils.StringToBool(options.Environment["COMPOSE_CONVERT_WINDOWS_PATHS"])
		}
	}
}

// save listeners from ProjectOptions (compose) to loader.Options
func withListeners(options *ProjectOptions) func(*loader.Options) {
	return func(opts *loader.Options) {
		opts.Listeners = append(opts.Listeners, options.Listeners...)
	}
}

// getConfigPaths retrieves the config files for project based on project options
func (o *ProjectOptions) getConfigPaths() ([]string, error) {
	if len(o.ConfigPaths) != 0 {
		return absolutePaths(o.ConfigPaths)
	}
	return nil, fmt.Errorf("no configuration file provided: %w", errdefs.ErrNotFound)
}

func findFiles(names []string, pwd string) []string {
	candidates := []string{}
	for _, n := range names {
		f := filepath.Join(pwd, n)
		if _, err := os.Stat(f); err == nil {
			candidates = append(candidates, f)
		}
	}
	return candidates
}

func absolutePaths(p []string) ([]string, error) {
	var paths []string
	for _, f := range p {
		if f == "-" {
			paths = append(paths, f)
			continue
		}
		abs, err := filepath.Abs(f)
		if err != nil {
			return nil, err
		}
		f = abs
		if _, err := os.Stat(f); err != nil {
			return nil, err
		}
		paths = append(paths, f)
	}
	return paths, nil
}
