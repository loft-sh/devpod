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
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	paths "path"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/consts"
	interp "github.com/compose-spec/compose-go/interpolation"
	"github.com/compose-spec/compose-go/schema"
	"github.com/compose-spec/compose-go/template"
	"github.com/compose-spec/compose-go/types"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Options supported by Load
type Options struct {
	// Skip schema validation
	SkipValidation bool
	// Skip interpolation
	SkipInterpolation bool
	// Skip normalization
	SkipNormalization bool
	// Resolve paths
	ResolvePaths bool
	// Convert Windows paths
	ConvertWindowsPaths bool
	// Skip consistency check
	SkipConsistencyCheck bool
	// Skip extends
	SkipExtends bool
	// SkipInclude will ignore `include` and only load model from file(s) set by ConfigDetails
	SkipInclude bool
	// SkipResolveEnvironment will ignore computing `environment` for services
	SkipResolveEnvironment bool
	// Interpolation options
	Interpolate *interp.Options
	// Discard 'env_file' entries after resolving to 'environment' section
	discardEnvFiles bool
	// Set project projectName
	projectName string
	// Indicates when the projectName was imperatively set or guessed from path
	projectNameImperativelySet bool
	// Profiles set profiles to enable
	Profiles []string
	// ResourceLoaders manages support for remote resources
	ResourceLoaders []ResourceLoader
}

// ResourceLoader is a plugable remote resource resolver
type ResourceLoader interface {
	// Accept returns `true` is the resource reference matches ResourceLoader supported protocol(s)
	Accept(path string) bool
	// Load returns the path to a local copy of remote resource identified by `path`.
	Load(ctx context.Context, path string) (string, error)
}

func (o *Options) clone() *Options {
	return &Options{
		SkipValidation:             o.SkipValidation,
		SkipInterpolation:          o.SkipInterpolation,
		SkipNormalization:          o.SkipNormalization,
		ResolvePaths:               o.ResolvePaths,
		ConvertWindowsPaths:        o.ConvertWindowsPaths,
		SkipConsistencyCheck:       o.SkipConsistencyCheck,
		SkipExtends:                o.SkipExtends,
		SkipInclude:                o.SkipInclude,
		Interpolate:                o.Interpolate,
		discardEnvFiles:            o.discardEnvFiles,
		projectName:                o.projectName,
		projectNameImperativelySet: o.projectNameImperativelySet,
		Profiles:                   o.Profiles,
		ResourceLoaders:            o.ResourceLoaders,
	}
}

func (o *Options) SetProjectName(name string, imperativelySet bool) {
	o.projectName = name
	o.projectNameImperativelySet = imperativelySet
}

func (o Options) GetProjectName() (string, bool) {
	return o.projectName, o.projectNameImperativelySet
}

// serviceRef identifies a reference to a service. It's used to detect cyclic
// references in "extends".
type serviceRef struct {
	filename string
	service  string
}

type cycleTracker struct {
	loaded []serviceRef
}

func (ct *cycleTracker) Add(filename, service string) error {
	toAdd := serviceRef{filename: filename, service: service}
	for _, loaded := range ct.loaded {
		if toAdd == loaded {
			// Create an error message of the form:
			// Circular reference:
			//   service-a in docker-compose.yml
			//   extends service-b in docker-compose.yml
			//   extends service-a in docker-compose.yml
			errLines := []string{
				"Circular reference:",
				fmt.Sprintf("  %s in %s", ct.loaded[0].service, ct.loaded[0].filename),
			}
			for _, service := range append(ct.loaded[1:], toAdd) {
				errLines = append(errLines, fmt.Sprintf("  extends %s in %s", service.service, service.filename))
			}

			return errors.New(strings.Join(errLines, "\n"))
		}
	}

	ct.loaded = append(ct.loaded, toAdd)
	return nil
}

// WithDiscardEnvFiles sets the Options to discard the `env_file` section after resolving to
// the `environment` section
func WithDiscardEnvFiles(opts *Options) {
	opts.discardEnvFiles = true
}

// WithSkipValidation sets the Options to skip validation when loading sections
func WithSkipValidation(opts *Options) {
	opts.SkipValidation = true
}

// WithProfiles sets profiles to be activated
func WithProfiles(profiles []string) func(*Options) {
	return func(opts *Options) {
		opts.Profiles = profiles
	}
}

// ParseYAML reads the bytes from a file, parses the bytes into a mapping
// structure, and returns it.
func ParseYAML(source []byte) (map[string]interface{}, error) {
	r := bytes.NewReader(source)
	decoder := yaml.NewDecoder(r)
	m, _, err := parseYAML(decoder)
	return m, err
}

// PostProcessor is used to tweak compose model based on metadata extracted during yaml Unmarshal phase
// that hardly can be implemented using go-yaml and mapstructure
type PostProcessor interface {
	yaml.Unmarshaler

	// Apply changes to compose model based on recorder metadata
	Apply(config *types.Config) error
}

func parseYAML(decoder *yaml.Decoder) (map[string]interface{}, PostProcessor, error) {
	var cfg interface{}
	processor := ResetProcessor{target: &cfg}

	if err := decoder.Decode(&processor); err != nil {
		return nil, nil, err
	}
	stringMap, ok := cfg.(map[string]interface{})
	if ok {
		converted, err := convertToStringKeysRecursive(stringMap, "")
		if err != nil {
			return nil, nil, err
		}
		return converted.(map[string]interface{}), &processor, nil
	}
	cfgMap, ok := cfg.(map[interface{}]interface{})
	if !ok {
		return nil, nil, errors.Errorf("Top-level object must be a mapping")
	}
	converted, err := convertToStringKeysRecursive(cfgMap, "")
	if err != nil {
		return nil, nil, err
	}
	return converted.(map[string]interface{}), &processor, nil
}

// Load reads a ConfigDetails and returns a fully loaded configuration.
// Deprecated: use LoadWithContext.
func Load(configDetails types.ConfigDetails, options ...func(*Options)) (*types.Project, error) {
	return LoadWithContext(context.Background(), configDetails, options...)
}

// LoadWithContext reads a ConfigDetails and returns a fully loaded configuration
func LoadWithContext(ctx context.Context, configDetails types.ConfigDetails, options ...func(*Options)) (*types.Project, error) {
	if len(configDetails.ConfigFiles) < 1 {
		return nil, errors.Errorf("No files specified")
	}

	opts := &Options{
		Interpolate: &interp.Options{
			Substitute:      template.Substitute,
			LookupValue:     configDetails.LookupEnv,
			TypeCastMapping: interpolateTypeCastMapping,
		},
		ResolvePaths: true,
	}

	for _, op := range options {
		op(opts)
	}

	projectName, err := projectName(configDetails, opts)
	if err != nil {
		return nil, err
	}
	opts.projectName = projectName

	// TODO(milas): this should probably ALWAYS set (overriding any existing)
	if _, ok := configDetails.Environment[consts.ComposeProjectName]; !ok && projectName != "" {
		if configDetails.Environment == nil {
			configDetails.Environment = map[string]string{}
		}
		configDetails.Environment[consts.ComposeProjectName] = projectName
	}

	return load(ctx, configDetails, opts, nil)
}

func load(ctx context.Context, configDetails types.ConfigDetails, opts *Options, loaded []string) (*types.Project, error) {
	var model *types.Config

	mainFile := configDetails.ConfigFiles[0].Filename
	for _, f := range loaded {
		if f == mainFile {
			loaded = append(loaded, mainFile)
			return nil, errors.Errorf("include cycle detected:\n%s\n include %s", loaded[0], strings.Join(loaded[1:], "\n include "))
		}
	}
	loaded = append(loaded, mainFile)

	includeRefs := make(map[string][]types.IncludeConfig)
	for _, file := range configDetails.ConfigFiles {
		var postProcessor PostProcessor
		configDict := file.Config

		processYaml := func() error {
			if !opts.SkipValidation {
				if err := schema.Validate(configDict); err != nil {
					return fmt.Errorf("validating %s: %w", file.Filename, err)
				}
			}

			configDict = groupXFieldsIntoExtensions(configDict)

			cfg, err := loadSections(ctx, file.Filename, configDict, configDetails, opts)
			if err != nil {
				return err
			}

			if !opts.SkipInclude {
				var included map[string][]types.IncludeConfig
				cfg, included, err = loadInclude(ctx, file.Filename, configDetails, cfg, opts, loaded)
				if err != nil {
					return err
				}
				for k, v := range included {
					includeRefs[k] = append(includeRefs[k], v...)
				}
			}

			if model == nil {
				model = cfg
			} else {
				merged, err := merge([]*types.Config{model, cfg})
				if err != nil {
					return err
				}
				model = merged
			}
			if postProcessor != nil {
				err = postProcessor.Apply(model)
				if err != nil {
					return err
				}
			}
			return nil
		}

		if configDict == nil {
			if len(file.Content) == 0 {
				content, err := os.ReadFile(file.Filename)
				if err != nil {
					return nil, err
				}
				file.Content = content
			}

			r := bytes.NewReader(file.Content)
			decoder := yaml.NewDecoder(r)
			for {
				dict, p, err := parseConfig(decoder, opts)
				if err != nil {
					if err != io.EOF {
						return nil, fmt.Errorf("parsing %s: %w", file.Filename, err)
					}
					break
				}
				configDict = dict
				postProcessor = p

				if err := processYaml(); err != nil {
					return nil, err
				}
			}
		} else {
			if err := processYaml(); err != nil {
				return nil, err
			}
		}
	}

	if model == nil {
		return nil, errors.New("empty compose file")
	}

	project := &types.Project{
		Name:        opts.projectName,
		WorkingDir:  configDetails.WorkingDir,
		Services:    model.Services,
		Networks:    model.Networks,
		Volumes:     model.Volumes,
		Secrets:     model.Secrets,
		Configs:     model.Configs,
		Environment: configDetails.Environment,
		Extensions:  model.Extensions,
	}

	if len(includeRefs) != 0 {
		project.IncludeReferences = includeRefs
	}

	if !opts.SkipNormalization {
		err := Normalize(project)
		if err != nil {
			return nil, err
		}
	}

	if opts.ResolvePaths {
		err := ResolveRelativePaths(project)
		if err != nil {
			return nil, err
		}
	}

	if opts.ConvertWindowsPaths {
		for i, service := range project.Services {
			for j, volume := range service.Volumes {
				service.Volumes[j] = convertVolumePath(volume)
			}
			project.Services[i] = service
		}
	}

	if !opts.SkipConsistencyCheck {
		err := checkConsistency(project)
		if err != nil {
			return nil, err
		}
	}

	project.ApplyProfiles(opts.Profiles)

	if !opts.SkipResolveEnvironment {
		err := project.ResolveServicesEnvironment(opts.discardEnvFiles)
		if err != nil {
			return nil, err
		}
	}

	return project, nil
}

func InvalidProjectNameErr(v string) error {
	return fmt.Errorf(
		"invalid project name %q: must consist only of lowercase alphanumeric characters, hyphens, and underscores as well as start with a letter or number",
		v,
	)
}

// projectName determines the canonical name to use for the project considering
// the loader Options as well as `name` fields in Compose YAML fields (which
// also support interpolation).
//
// TODO(milas): restructure loading so that we don't need to re-parse the YAML
// here, as it's both wasteful and makes this code error-prone.
func projectName(details types.ConfigDetails, opts *Options) (string, error) {
	projectName, projectNameImperativelySet := opts.GetProjectName()

	// if user did NOT provide a name explicitly, then see if one is defined
	// in any of the config files
	if !projectNameImperativelySet {
		var pjNameFromConfigFile string
		for _, configFile := range details.ConfigFiles {
			yml, err := ParseYAML(configFile.Content)
			if err != nil {
				// HACK: the way that loading is currently structured, this is
				// a duplicative parse just for the `name`. if it fails, we
				// give up but don't return the error, knowing that it'll get
				// caught downstream for us
				return "", nil
			}
			if val, ok := yml["name"]; ok && val != "" {
				sVal, ok := val.(string)
				if !ok {
					// HACK: see above - this is a temporary parsed version
					// that hasn't been schema-validated, but we don't want
					// to be the ones to actually report that, so give up,
					// knowing that it'll get caught downstream for us
					return "", nil
				}
				pjNameFromConfigFile = sVal
			}
		}
		if !opts.SkipInterpolation {
			interpolated, err := interp.Interpolate(
				map[string]interface{}{"name": pjNameFromConfigFile},
				*opts.Interpolate,
			)
			if err != nil {
				return "", err
			}
			pjNameFromConfigFile = interpolated["name"].(string)
		}
		pjNameFromConfigFile = NormalizeProjectName(pjNameFromConfigFile)
		if pjNameFromConfigFile != "" {
			projectName = pjNameFromConfigFile
		}
	}

	if projectName == "" {
		return "", errors.New("project name must not be empty")
	}

	if NormalizeProjectName(projectName) != projectName {
		return "", InvalidProjectNameErr(projectName)
	}

	return projectName, nil
}

func NormalizeProjectName(s string) string {
	r := regexp.MustCompile("[a-z0-9_-]")
	s = strings.ToLower(s)
	s = strings.Join(r.FindAllString(s, -1), "")
	return strings.TrimLeft(s, "_-")
}

func parseConfig(decoder *yaml.Decoder, opts *Options) (map[string]interface{}, PostProcessor, error) {
	yml, postProcessor, err := parseYAML(decoder)
	if err != nil {
		return nil, nil, err
	}
	if !opts.SkipInterpolation {
		interpolated, err := interp.Interpolate(yml, *opts.Interpolate)
		return interpolated, postProcessor, err
	}
	return yml, postProcessor, err
}

const extensions = "#extensions" // Using # prefix, we prevent risk to conflict with an actual yaml key

func groupXFieldsIntoExtensions(dict map[string]interface{}) map[string]interface{} {
	extras := map[string]interface{}{}
	for key, value := range dict {
		if strings.HasPrefix(key, "x-") {
			extras[key] = value
			delete(dict, key)
		}
		if d, ok := value.(map[string]interface{}); ok {
			dict[key] = groupXFieldsIntoExtensions(d)
		}
	}
	if len(extras) > 0 {
		dict[extensions] = extras
	}
	return dict
}

func loadSections(ctx context.Context, filename string, config map[string]interface{}, configDetails types.ConfigDetails, opts *Options) (*types.Config, error) {
	var err error
	cfg := types.Config{
		Filename: filename,
	}
	name := ""
	if n, ok := config["name"]; ok {
		name, ok = n.(string)
		if !ok {
			return nil, errors.New("project name must be a string")
		}
	}
	cfg.Name = name
	cfg.Services, err = LoadServices(ctx, filename, getSection(config, "services"), configDetails.WorkingDir, configDetails.LookupEnv, opts)
	if err != nil {
		return nil, err
	}
	cfg.Networks, err = LoadNetworks(getSection(config, "networks"))
	if err != nil {
		return nil, err
	}
	cfg.Volumes, err = LoadVolumes(getSection(config, "volumes"))
	if err != nil {
		return nil, err
	}
	cfg.Secrets, err = LoadSecrets(getSection(config, "secrets"))
	if err != nil {
		return nil, err
	}
	cfg.Configs, err = LoadConfigObjs(getSection(config, "configs"))
	if err != nil {
		return nil, err
	}
	cfg.Include, err = LoadIncludeConfig(getSequence(config, "include"))
	if err != nil {
		return nil, err
	}
	extensions := getSection(config, extensions)
	if len(extensions) > 0 {
		cfg.Extensions = extensions
	}
	return &cfg, nil
}

func getSection(config map[string]interface{}, key string) map[string]interface{} {
	section, ok := config[key]
	if !ok {
		return make(map[string]interface{})
	}
	return section.(map[string]interface{})
}

func getSequence(config map[string]interface{}, key string) []interface{} {
	section, ok := config[key]
	if !ok {
		return make([]interface{}, 0)
	}
	return section.([]interface{})
}

// ForbiddenPropertiesError is returned when there are properties in the Compose
// file that are forbidden.
type ForbiddenPropertiesError struct {
	Properties map[string]string
}

func (e *ForbiddenPropertiesError) Error() string {
	return "Configuration contains forbidden properties"
}

// Transform converts the source into the target struct with compose types transformer
// and the specified transformers if any.
func Transform(source interface{}, target interface{}, additionalTransformers ...Transformer) error {
	data := mapstructure.Metadata{}
	config := &mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			createTransformHook(additionalTransformers...),
			decoderHook),
		Result:   target,
		TagName:  "yaml",
		Metadata: &data,
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(source)
}

// TransformerFunc defines a function to perform the actual transformation
type TransformerFunc func(interface{}) (interface{}, error)

// Transformer defines a map to type transformer
type Transformer struct {
	TypeOf reflect.Type
	Func   TransformerFunc
}

func createTransformHook(additionalTransformers ...Transformer) mapstructure.DecodeHookFuncType {
	transforms := map[reflect.Type]func(interface{}) (interface{}, error){
		reflect.TypeOf(types.External{}):                         transformExternal,
		reflect.TypeOf(types.Options{}):                          transformOptions,
		reflect.TypeOf(types.UlimitsConfig{}):                    transformUlimits,
		reflect.TypeOf([]types.ServicePortConfig{}):              transformServicePort,
		reflect.TypeOf(types.ServiceSecretConfig{}):              transformFileReferenceConfig,
		reflect.TypeOf(types.ServiceConfigObjConfig{}):           transformFileReferenceConfig,
		reflect.TypeOf(map[string]*types.ServiceNetworkConfig{}): transformServiceNetworkMap,
		reflect.TypeOf(types.Mapping{}):                          transformMappingOrListFunc("=", false),
		reflect.TypeOf(types.MappingWithEquals{}):                transformMappingOrListFunc("=", true),
		reflect.TypeOf(types.MappingWithColon{}):                 transformMappingOrListFunc(":", false),
		reflect.TypeOf(types.HostsList{}):                        transformMappingOrListFunc(":", false),
		reflect.TypeOf(types.ServiceVolumeConfig{}):              transformServiceVolumeConfig,
		reflect.TypeOf(types.BuildConfig{}):                      transformBuildConfig,
		reflect.TypeOf(types.DependsOnConfig{}):                  transformDependsOnConfig,
		reflect.TypeOf(types.ExtendsConfig{}):                    transformExtendsConfig,
		reflect.TypeOf(types.SSHConfig{}):                        transformSSHConfig,
		reflect.TypeOf(types.IncludeConfig{}):                    transformIncludeConfig,
	}

	for _, transformer := range additionalTransformers {
		transforms[transformer.TypeOf] = transformer.Func
	}

	return func(_ reflect.Type, target reflect.Type, data interface{}) (interface{}, error) {
		transform, ok := transforms[target]
		if !ok {
			return data, nil
		}
		return transform(data)
	}
}

// keys need to be converted to strings for jsonschema
func convertToStringKeysRecursive(value interface{}, keyPrefix string) (interface{}, error) {
	if mapping, ok := value.(map[string]interface{}); ok {
		for key, entry := range mapping {
			var newKeyPrefix string
			if keyPrefix == "" {
				newKeyPrefix = key
			} else {
				newKeyPrefix = fmt.Sprintf("%s.%s", keyPrefix, key)
			}
			convertedEntry, err := convertToStringKeysRecursive(entry, newKeyPrefix)
			if err != nil {
				return nil, err
			}
			mapping[key] = convertedEntry
		}
		return mapping, nil
	}
	if mapping, ok := value.(map[interface{}]interface{}); ok {
		dict := make(map[string]interface{})
		for key, entry := range mapping {
			str, ok := key.(string)
			if !ok {
				return nil, formatInvalidKeyError(keyPrefix, key)
			}
			var newKeyPrefix string
			if keyPrefix == "" {
				newKeyPrefix = str
			} else {
				newKeyPrefix = fmt.Sprintf("%s.%s", keyPrefix, str)
			}
			convertedEntry, err := convertToStringKeysRecursive(entry, newKeyPrefix)
			if err != nil {
				return nil, err
			}
			dict[str] = convertedEntry
		}
		return dict, nil
	}
	if list, ok := value.([]interface{}); ok {
		var convertedList []interface{}
		for index, entry := range list {
			newKeyPrefix := fmt.Sprintf("%s[%d]", keyPrefix, index)
			convertedEntry, err := convertToStringKeysRecursive(entry, newKeyPrefix)
			if err != nil {
				return nil, err
			}
			convertedList = append(convertedList, convertedEntry)
		}
		return convertedList, nil
	}
	return value, nil
}

func formatInvalidKeyError(keyPrefix string, key interface{}) error {
	var location string
	if keyPrefix == "" {
		location = "at top level"
	} else {
		location = fmt.Sprintf("in %s", keyPrefix)
	}
	return errors.Errorf("Non-string key %s: %#v", location, key)
}

// LoadServices produces a ServiceConfig map from a compose file Dict
// the servicesDict is not validated if directly used. Use Load() to enable validation
func LoadServices(ctx context.Context, filename string, servicesDict map[string]interface{}, workingDir string, lookupEnv template.Mapping, opts *Options) ([]types.ServiceConfig, error) {
	var services []types.ServiceConfig

	x, ok := servicesDict[extensions]
	if ok {
		// as a top-level attribute, "services" doesn't support extensions, and a service can be named `x-foo`
		for k, v := range x.(map[string]interface{}) {
			servicesDict[k] = v
		}
		delete(servicesDict, extensions)
	}

	for name := range servicesDict {
		serviceConfig, err := loadServiceWithExtends(ctx, filename, name, servicesDict, workingDir, lookupEnv, opts, &cycleTracker{})
		if err != nil {
			return nil, err
		}

		services = append(services, *serviceConfig)
	}

	return services, nil
}

func loadServiceWithExtends(ctx context.Context, filename, name string, servicesDict map[string]interface{}, workingDir string, lookupEnv template.Mapping, opts *Options, ct *cycleTracker) (*types.ServiceConfig, error) {
	if err := ct.Add(filename, name); err != nil {
		return nil, err
	}

	target, ok := servicesDict[name]
	if !ok {
		return nil, fmt.Errorf("cannot extend service %q in %s: service not found", name, filename)
	}

	if target == nil {
		target = map[string]interface{}{}
	}

	serviceConfig, err := LoadService(name, target.(map[string]interface{}))
	if err != nil {
		return nil, err
	}

	if serviceConfig.Extends != nil && !opts.SkipExtends {
		baseServiceName := serviceConfig.Extends.Service
		var baseService *types.ServiceConfig
		file := serviceConfig.Extends.File
		if file == "" {
			baseService, err = loadServiceWithExtends(ctx, filename, baseServiceName, servicesDict, workingDir, lookupEnv, opts, ct)
			if err != nil {
				return nil, err
			}
		} else {
			for _, loader := range opts.ResourceLoaders {
				if loader.Accept(file) {
					path, err := loader.Load(ctx, file)
					if err != nil {
						return nil, err
					}
					file = path
					break
				}
			}
			// Resolve the path to the imported file, and load it.
			baseFilePath := absPath(workingDir, file)

			b, err := os.ReadFile(baseFilePath)
			if err != nil {
				return nil, err
			}

			r := bytes.NewReader(b)
			decoder := yaml.NewDecoder(r)

			baseFile, _, err := parseConfig(decoder, opts)
			if err != nil {
				return nil, err
			}

			baseFileServices := getSection(baseFile, "services")
			baseService, err = loadServiceWithExtends(ctx, baseFilePath, baseServiceName, baseFileServices, filepath.Dir(baseFilePath), lookupEnv, opts, ct)
			if err != nil {
				return nil, err
			}

			// Make paths relative to the importing Compose file. Note that we
			// make the paths relative to `file` rather than `baseFilePath` so
			// that the resulting paths won't be absolute if `file` isn't an
			// absolute path.

			baseFileParent := filepath.Dir(file)
			ResolveServiceRelativePaths(baseFileParent, baseService)
		}

		serviceConfig, err = _merge(baseService, serviceConfig)
		if err != nil {
			return nil, err
		}
		serviceConfig.Extends = nil
	}

	return serviceConfig, nil
}

// LoadService produces a single ServiceConfig from a compose file Dict
// the serviceDict is not validated if directly used. Use Load() to enable validation
func LoadService(name string, serviceDict map[string]interface{}) (*types.ServiceConfig, error) {
	serviceConfig := &types.ServiceConfig{
		Scale: 1,
	}
	if err := Transform(serviceDict, serviceConfig); err != nil {
		return nil, err
	}
	serviceConfig.Name = name

	for i, volume := range serviceConfig.Volumes {
		if volume.Type != types.VolumeTypeBind {
			continue
		}
		if volume.Source == "" {
			return nil, errors.New(`invalid mount config for type "bind": field Source must not be empty`)
		}

		serviceConfig.Volumes[i] = volume
	}

	return serviceConfig, nil
}

// Windows paths, c:\\my\\path\\shiny, need to be changed to be compatible with
// the Engine. Volume paths are expected to be linux style /c/my/path/shiny/
func convertVolumePath(volume types.ServiceVolumeConfig) types.ServiceVolumeConfig {
	volumeName := strings.ToLower(filepath.VolumeName(volume.Source))
	if len(volumeName) != 2 {
		return volume
	}

	convertedSource := fmt.Sprintf("/%c%s", volumeName[0], volume.Source[len(volumeName):])
	convertedSource = strings.ReplaceAll(convertedSource, "\\", "/")

	volume.Source = convertedSource
	return volume
}

func resolveMaybeUnixPath(workingDir string, path string) string {
	filePath := expandUser(path)
	// Check if source is an absolute path (either Unix or Windows), to
	// handle a Windows client with a Unix daemon or vice-versa.
	//
	// Note that this is not required for Docker for Windows when specifying
	// a local Windows path, because Docker for Windows translates the Windows
	// path into a valid path within the VM.
	if !paths.IsAbs(filePath) && !isAbs(filePath) {
		filePath = absPath(workingDir, filePath)
	}
	return filePath
}

// TODO: make this more robust
func expandUser(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			logrus.Warn("cannot expand '~', because the environment lacks HOME")
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

func transformUlimits(data interface{}) (interface{}, error) {
	switch value := data.(type) {
	case int:
		return types.UlimitsConfig{Single: value}, nil
	case map[string]interface{}:
		ulimit := types.UlimitsConfig{}
		if v, ok := value["soft"]; ok {
			ulimit.Soft = v.(int)
		}
		if v, ok := value["hard"]; ok {
			ulimit.Hard = v.(int)
		}
		return ulimit, nil
	default:
		return data, errors.Errorf("invalid type %T for ulimits", value)
	}
}

// LoadNetworks produces a NetworkConfig map from a compose file Dict
// the source Dict is not validated if directly used. Use Load() to enable validation
func LoadNetworks(source map[string]interface{}) (map[string]types.NetworkConfig, error) {
	networks := make(map[string]types.NetworkConfig)
	err := Transform(source, &networks)
	if err != nil {
		return networks, err
	}
	for name, network := range networks {
		if !network.External.External {
			continue
		}
		switch {
		case network.External.Name != "":
			if network.Name != "" {
				return nil, errors.Errorf("network %s: network.external.name and network.name conflict; only use network.name", name)
			}
			logrus.Warnf("network %s: network.external.name is deprecated. Please set network.name with external: true", name)
			network.Name = network.External.Name
			network.External.Name = ""
		case network.Name == "":
			network.Name = name
		}
		networks[name] = network
	}
	return networks, nil
}

func externalVolumeError(volume, key string) error {
	return errors.Errorf(
		"conflicting parameters \"external\" and %q specified for volume %q",
		key, volume)
}

// LoadVolumes produces a VolumeConfig map from a compose file Dict
// the source Dict is not validated if directly used. Use Load() to enable validation
func LoadVolumes(source map[string]interface{}) (map[string]types.VolumeConfig, error) {
	volumes := make(map[string]types.VolumeConfig)
	if err := Transform(source, &volumes); err != nil {
		return volumes, err
	}

	for name, volume := range volumes {
		if !volume.External.External {
			continue
		}
		switch {
		case volume.Driver != "":
			return nil, externalVolumeError(name, "driver")
		case len(volume.DriverOpts) > 0:
			return nil, externalVolumeError(name, "driver_opts")
		case len(volume.Labels) > 0:
			return nil, externalVolumeError(name, "labels")
		case volume.External.Name != "":
			if volume.Name != "" {
				return nil, errors.Errorf("volume %s: volume.external.name and volume.name conflict; only use volume.name", name)
			}
			logrus.Warnf("volume %s: volume.external.name is deprecated in favor of volume.name", name)
			volume.Name = volume.External.Name
			volume.External.Name = ""
		case volume.Name == "":
			volume.Name = name
		}
		volumes[name] = volume
	}
	return volumes, nil
}

// LoadSecrets produces a SecretConfig map from a compose file Dict
// the source Dict is not validated if directly used. Use Load() to enable validation
func LoadSecrets(source map[string]interface{}) (map[string]types.SecretConfig, error) {
	secrets := make(map[string]types.SecretConfig)
	if err := Transform(source, &secrets); err != nil {
		return secrets, err
	}
	for name, secret := range secrets {
		obj, err := loadFileObjectConfig(name, "secret", types.FileObjectConfig(secret))
		if err != nil {
			return nil, err
		}
		secrets[name] = types.SecretConfig(obj)
	}
	return secrets, nil
}

// LoadConfigObjs produces a ConfigObjConfig map from a compose file Dict
// the source Dict is not validated if directly used. Use Load() to enable validation
func LoadConfigObjs(source map[string]interface{}) (map[string]types.ConfigObjConfig, error) {
	configs := make(map[string]types.ConfigObjConfig)
	if err := Transform(source, &configs); err != nil {
		return configs, err
	}
	for name, config := range configs {
		obj, err := loadFileObjectConfig(name, "config", types.FileObjectConfig(config))
		if err != nil {
			return nil, err
		}
		configs[name] = types.ConfigObjConfig(obj)
	}
	return configs, nil
}

func loadFileObjectConfig(name string, objType string, obj types.FileObjectConfig) (types.FileObjectConfig, error) {
	// if "external: true"
	switch {
	case obj.External.External:
		// handle deprecated external.name
		if obj.External.Name != "" {
			if obj.Name != "" {
				return obj, errors.Errorf("%[1]s %[2]s: %[1]s.external.name and %[1]s.name conflict; only use %[1]s.name", objType, name)
			}
			logrus.Warnf("%[1]s %[2]s: %[1]s.external.name is deprecated in favor of %[1]s.name", objType, name)
			obj.Name = obj.External.Name
			obj.External.Name = ""
		} else if obj.Name == "" {
			obj.Name = name
		}
		// if not "external: true"
	case obj.Driver != "":
		if obj.File != "" {
			return obj, errors.Errorf("%[1]s %[2]s: %[1]s.driver and %[1]s.file conflict; only use %[1]s.driver", objType, name)
		}
	}

	return obj, nil
}

var transformOptions TransformerFunc = func(data interface{}) (interface{}, error) {
	switch value := data.(type) {
	case map[string]interface{}:
		return toMapStringString(value, false), nil
	case map[string]string:
		return value, nil
	default:
		return data, errors.Errorf("invalid type %T for map[string]string", value)
	}
}

var transformExternal TransformerFunc = func(data interface{}) (interface{}, error) {
	switch value := data.(type) {
	case bool:
		return map[string]interface{}{"external": value}, nil
	case map[string]interface{}:
		return map[string]interface{}{"external": true, "name": value["name"]}, nil
	default:
		return data, errors.Errorf("invalid type %T for external", value)
	}
}

var transformServicePort TransformerFunc = func(data interface{}) (interface{}, error) {
	switch entries := data.(type) {
	case []interface{}:
		// We process the list instead of individual items here.
		// The reason is that one entry might be mapped to multiple ServicePortConfig.
		// Therefore we take an input of a list and return an output of a list.
		var ports []interface{}
		for _, entry := range entries {
			switch value := entry.(type) {
			case int:
				parsed, err := types.ParsePortConfig(fmt.Sprint(value))
				if err != nil {
					return data, err
				}
				for _, v := range parsed {
					ports = append(ports, v)
				}
			case string:
				parsed, err := types.ParsePortConfig(value)
				if err != nil {
					return data, err
				}
				for _, v := range parsed {
					ports = append(ports, v)
				}
			case map[string]interface{}:
				published := value["published"]
				if v, ok := published.(int); ok {
					value["published"] = strconv.Itoa(v)
				}
				ports = append(ports, groupXFieldsIntoExtensions(value))
			default:
				return data, errors.Errorf("invalid type %T for port", value)
			}
		}
		return ports, nil
	default:
		return data, errors.Errorf("invalid type %T for port", entries)
	}
}

var transformFileReferenceConfig TransformerFunc = func(data interface{}) (interface{}, error) {
	switch value := data.(type) {
	case string:
		return map[string]interface{}{"source": value}, nil
	case map[string]interface{}:
		if target, ok := value["target"]; ok {
			value["target"] = cleanTarget(target.(string))
		}
		return groupXFieldsIntoExtensions(value), nil
	default:
		return data, errors.Errorf("invalid type %T for secret", value)
	}
}

func cleanTarget(target string) string {
	if target == "" {
		return ""
	}
	return paths.Clean(target)
}

var transformBuildConfig TransformerFunc = func(data interface{}) (interface{}, error) {
	switch value := data.(type) {
	case string:
		return map[string]interface{}{"context": value}, nil
	case map[string]interface{}:
		return groupXFieldsIntoExtensions(data.(map[string]interface{})), nil
	default:
		return data, errors.Errorf("invalid type %T for service build", value)
	}
}

var transformDependsOnConfig TransformerFunc = func(data interface{}) (interface{}, error) {
	switch value := data.(type) {
	case []interface{}:
		transformed := map[string]interface{}{}
		for _, serviceIntf := range value {
			service, ok := serviceIntf.(string)
			if !ok {
				return data, errors.Errorf("invalid type %T for service depends_on element, expected string", value)
			}
			transformed[service] = map[string]interface{}{"condition": types.ServiceConditionStarted, "required": true}
		}
		return transformed, nil
	case map[string]interface{}:
		transformed := map[string]interface{}{}
		for service, val := range value {
			dependsConfigIntf, ok := val.(map[string]interface{})
			if !ok {
				return data, errors.Errorf("invalid type %T for service depends_on element", value)
			}
			if _, ok := dependsConfigIntf["required"]; !ok {
				dependsConfigIntf["required"] = true
			}
			transformed[service] = dependsConfigIntf
		}
		return groupXFieldsIntoExtensions(transformed), nil
	default:
		return data, errors.Errorf("invalid type %T for service depends_on", value)
	}
}

var transformExtendsConfig TransformerFunc = func(value interface{}) (interface{}, error) {
	switch value.(type) {
	case string:
		return map[string]interface{}{"service": value}, nil
	case map[string]interface{}:
		return value, nil
	default:
		return value, errors.Errorf("invalid type %T for extends", value)
	}
}

var transformServiceVolumeConfig TransformerFunc = func(data interface{}) (interface{}, error) {
	switch value := data.(type) {
	case string:
		volume, err := ParseVolume(value)
		volume.Target = cleanTarget(volume.Target)
		return volume, err
	case map[string]interface{}:
		data := groupXFieldsIntoExtensions(data.(map[string]interface{}))
		if target, ok := data["target"]; ok {
			data["target"] = cleanTarget(target.(string))
		}
		return data, nil
	default:
		return data, errors.Errorf("invalid type %T for service volume", value)
	}
}

var transformServiceNetworkMap TransformerFunc = func(value interface{}) (interface{}, error) {
	if list, ok := value.([]interface{}); ok {
		mapValue := map[interface{}]interface{}{}
		for _, name := range list {
			mapValue[name] = nil
		}
		return mapValue, nil
	}
	return value, nil
}

var transformSSHConfig TransformerFunc = func(data interface{}) (interface{}, error) {
	switch value := data.(type) {
	case map[string]interface{}:
		var result []types.SSHKey
		for key, val := range value {
			if val == nil {
				val = ""
			}
			result = append(result, types.SSHKey{ID: key, Path: val.(string)})
		}
		return result, nil
	case []interface{}:
		var result []types.SSHKey
		for _, v := range value {
			key, val := transformValueToMapEntry(v.(string), "=", false)
			result = append(result, types.SSHKey{ID: key, Path: val.(string)})
		}
		return result, nil
	case string:
		return ParseShortSSHSyntax(value)
	}
	return nil, errors.Errorf("expected a sting, map or a list, got %T: %#v", data, data)
}

// ParseShortSSHSyntax parse short syntax for SSH authentications
func ParseShortSSHSyntax(value string) ([]types.SSHKey, error) {
	if value == "" {
		value = "default"
	}
	key, val := transformValueToMapEntry(value, "=", false)
	result := []types.SSHKey{{ID: key, Path: val.(string)}}
	return result, nil
}

func transformMappingOrListFunc(sep string, allowNil bool) TransformerFunc {
	return func(data interface{}) (interface{}, error) {
		return transformMappingOrList(data, sep, allowNil)
	}
}

func transformMappingOrList(mappingOrList interface{}, sep string, allowNil bool) (interface{}, error) {
	switch value := mappingOrList.(type) {
	case map[string]interface{}:
		return toMapStringString(value, allowNil), nil
	case []interface{}:
		result := make(map[string]interface{})
		for _, value := range value {
			key, val := transformValueToMapEntry(value.(string), sep, allowNil)
			result[key] = val
		}
		return result, nil
	}
	return nil, errors.Errorf("expected a map or a list, got %T: %#v", mappingOrList, mappingOrList)
}

func transformValueToMapEntry(value string, separator string, allowNil bool) (string, interface{}) {
	parts := strings.SplitN(value, separator, 2)
	key := parts[0]
	switch {
	case len(parts) == 1 && allowNil:
		return key, nil
	case len(parts) == 1 && !allowNil:
		return key, ""
	default:
		return key, parts[1]
	}
}

func toMapStringString(value map[string]interface{}, allowNil bool) map[string]interface{} {
	output := make(map[string]interface{})
	for key, value := range value {
		output[key] = toString(value, allowNil)
	}
	return output
}

func toString(value interface{}, allowNil bool) interface{} {
	switch {
	case value != nil:
		return fmt.Sprint(value)
	case allowNil:
		return nil
	default:
		return ""
	}
}
