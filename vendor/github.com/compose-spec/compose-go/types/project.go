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

package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/compose-spec/compose-go/dotenv"
	"github.com/compose-spec/compose-go/utils"
	"github.com/distribution/reference"
	godigest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

// Project is the result of loading a set of compose files
type Project struct {
	Name       string     `yaml:"name,omitempty" json:"name,omitempty"`
	WorkingDir string     `yaml:"-" json:"-"`
	Services   Services   `yaml:"services" json:"services"`
	Networks   Networks   `yaml:"networks,omitempty" json:"networks,omitempty"`
	Volumes    Volumes    `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Secrets    Secrets    `yaml:"secrets,omitempty" json:"secrets,omitempty"`
	Configs    Configs    `yaml:"configs,omitempty" json:"configs,omitempty"`
	Extensions Extensions `yaml:"#extensions,inline" json:"-"` // https://github.com/golang/go/issues/6213

	// IncludeReferences is keyed by Compose YAML filename and contains config for
	// other Compose YAML files it directly triggered a load of via `include`.
	//
	// Note: this is
	IncludeReferences map[string][]IncludeConfig `yaml:"-" json:"-"`
	ComposeFiles      []string                   `yaml:"-" json:"-"`
	Environment       Mapping                    `yaml:"-" json:"-"`

	// DisabledServices track services which have been disable as profile is not active
	DisabledServices Services `yaml:"-" json:"-"`
	Profiles         []string `yaml:"-" json:"-"`
}

// ServiceNames return names for all services in this Compose config
func (p *Project) ServiceNames() []string {
	var names []string
	for _, s := range p.Services {
		names = append(names, s.Name)
	}
	sort.Strings(names)
	return names
}

// VolumeNames return names for all volumes in this Compose config
func (p *Project) VolumeNames() []string {
	var names []string
	for k := range p.Volumes {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// NetworkNames return names for all volumes in this Compose config
func (p *Project) NetworkNames() []string {
	var names []string
	for k := range p.Networks {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// SecretNames return names for all secrets in this Compose config
func (p *Project) SecretNames() []string {
	var names []string
	for k := range p.Secrets {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// ConfigNames return names for all configs in this Compose config
func (p *Project) ConfigNames() []string {
	var names []string
	for k := range p.Configs {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// GetServices retrieve services by names, or return all services if no name specified
func (p *Project) GetServices(names ...string) (Services, error) {
	services, servicesNotFound := p.getServicesByNames(names...)
	if len(servicesNotFound) > 0 {
		return services, fmt.Errorf("no such service: %s", servicesNotFound[0])
	}
	return services, nil
}

func (p *Project) getServicesByNames(names ...string) (Services, []string) {
	if len(names) == 0 {
		return p.Services, nil
	}
	services := Services{}
	var servicesNotFound []string
	for _, name := range names {
		var serviceConfig *ServiceConfig
		for _, s := range p.Services {
			if s.Name == name {
				serviceConfig = &s
				break
			}
		}
		if serviceConfig == nil {
			servicesNotFound = append(servicesNotFound, name)
			continue
		}
		services = append(services, *serviceConfig)
	}
	return services, servicesNotFound
}

// GetDisabledService retrieve disabled service by name
func (p Project) GetDisabledService(name string) (ServiceConfig, error) {
	for _, config := range p.DisabledServices {
		if config.Name == name {
			return config, nil
		}
	}
	return ServiceConfig{}, fmt.Errorf("no such service: %s", name)
}

// GetService retrieve a specific service by name
func (p *Project) GetService(name string) (ServiceConfig, error) {
	services, err := p.GetServices(name)
	if err != nil {
		return ServiceConfig{}, err
	}
	if len(services) == 0 {
		return ServiceConfig{}, fmt.Errorf("no such service: %s", name)
	}
	return services[0], nil
}

func (p *Project) AllServices() Services {
	var all Services
	all = append(all, p.Services...)
	all = append(all, p.DisabledServices...)
	return all
}

type ServiceFunc func(service ServiceConfig) error

// WithServices run ServiceFunc on each service and dependencies according to DependencyPolicy
func (p *Project) WithServices(names []string, fn ServiceFunc, options ...DependencyOption) error {
	if len(options) == 0 {
		// backward compatibility
		options = []DependencyOption{IncludeDependencies}
	}
	return p.withServices(names, fn, map[string]bool{}, options, map[string]ServiceDependency{})
}

func (p *Project) withServices(names []string, fn ServiceFunc, seen map[string]bool, options []DependencyOption, dependencies map[string]ServiceDependency) error {
	services, servicesNotFound := p.getServicesByNames(names...)
	if len(servicesNotFound) > 0 {
		for _, serviceNotFound := range servicesNotFound {
			if dependency, ok := dependencies[serviceNotFound]; !ok || dependency.Required {
				return fmt.Errorf("no such service: %s", serviceNotFound)
			}
		}
	}
	for _, service := range services {
		if seen[service.Name] {
			continue
		}
		seen[service.Name] = true
		var dependencies map[string]ServiceDependency
		for _, policy := range options {
			switch policy {
			case IncludeDependents:
				dependencies = utils.MapsAppend(dependencies, p.dependentsForService(service))
			case IncludeDependencies:
				dependencies = utils.MapsAppend(dependencies, service.DependsOn)
			case IgnoreDependencies:
				// Noop
			default:
				return fmt.Errorf("unsupported dependency policy %d", policy)
			}
		}
		if len(dependencies) > 0 {
			err := p.withServices(utils.MapKeys(dependencies), fn, seen, options, dependencies)
			if err != nil {
				return err
			}
		}
		if err := fn(service); err != nil {
			return err
		}
	}
	return nil
}

func (p *Project) GetDependentsForService(s ServiceConfig) []string {
	return utils.MapKeys(p.dependentsForService(s))
}

func (p *Project) dependentsForService(s ServiceConfig) map[string]ServiceDependency {
	dependent := make(map[string]ServiceDependency)
	for _, service := range p.Services {
		for name, dependency := range service.DependsOn {
			if name == s.Name {
				dependent[service.Name] = dependency
			}
		}
	}
	return dependent
}

// RelativePath resolve a relative path based project's working directory
func (p *Project) RelativePath(path string) string {
	if path[0] == '~' {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[1:])
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(p.WorkingDir, path)
}

// HasProfile return true if service has no profile declared or has at least one profile matching
func (s ServiceConfig) HasProfile(profiles []string) bool {
	if len(s.Profiles) == 0 {
		return true
	}
	for _, p := range profiles {
		for _, sp := range s.Profiles {
			if sp == p {
				return true
			}
		}
	}
	return false
}

// GetProfiles retrieve the profiles implicitly enabled by explicitly targeting selected services
func (s Services) GetProfiles() []string {
	set := map[string]struct{}{}
	for _, service := range s {
		for _, p := range service.Profiles {
			set[p] = struct{}{}
		}
	}
	var profiles []string
	for k := range set {
		profiles = append(profiles, k)
	}
	return profiles
}

// ApplyProfiles disables service which don't match selected profiles
func (p *Project) ApplyProfiles(profiles []string) {
	for _, p := range profiles {
		if p == "*" {
			return
		}
	}
	var enabled, disabled Services
	for _, service := range p.AllServices() {
		if service.HasProfile(profiles) {
			enabled = append(enabled, service)
		} else {
			disabled = append(disabled, service)
		}
	}
	p.Services = enabled
	p.DisabledServices = disabled
	p.Profiles = profiles
}

// EnableServices ensure services are enabled and activate profiles accordingly
func (p *Project) EnableServices(names ...string) error {
	if len(names) == 0 {
		return nil
	}
	var enabled []string
	for _, name := range names {
		_, err := p.GetService(name)
		if err == nil {
			// already enabled
			continue
		}
		def, err := p.GetDisabledService(name)
		if err != nil {
			return err
		}
		enabled = append(enabled, def.Profiles...)
	}

	profiles := p.Profiles
PROFILES:
	for _, profile := range enabled {
		for _, p := range profiles {
			if p == profile {
				continue PROFILES
			}
		}
		profiles = append(profiles, profile)
	}
	p.ApplyProfiles(profiles)

	return p.ResolveServicesEnvironment(true)
}

// WithoutUnnecessaryResources drops networks/volumes/secrets/configs that are not referenced by active services
func (p *Project) WithoutUnnecessaryResources() {
	requiredNetworks := map[string]struct{}{}
	requiredVolumes := map[string]struct{}{}
	requiredSecrets := map[string]struct{}{}
	requiredConfigs := map[string]struct{}{}
	for _, s := range p.Services {
		for k := range s.Networks {
			requiredNetworks[k] = struct{}{}
		}
		for _, v := range s.Volumes {
			if v.Type != VolumeTypeVolume || v.Source == "" {
				continue
			}
			requiredVolumes[v.Source] = struct{}{}
		}
		for _, v := range s.Secrets {
			requiredSecrets[v.Source] = struct{}{}
		}
		if s.Build != nil {
			for _, v := range s.Build.Secrets {
				requiredSecrets[v.Source] = struct{}{}
			}
		}
		for _, v := range s.Configs {
			requiredConfigs[v.Source] = struct{}{}
		}
	}

	networks := Networks{}
	for k := range requiredNetworks {
		if value, ok := p.Networks[k]; ok {
			networks[k] = value
		}
	}
	p.Networks = networks

	volumes := Volumes{}
	for k := range requiredVolumes {
		if value, ok := p.Volumes[k]; ok {
			volumes[k] = value
		}
	}
	p.Volumes = volumes

	secrets := Secrets{}
	for k := range requiredSecrets {
		if value, ok := p.Secrets[k]; ok {
			secrets[k] = value
		}
	}
	p.Secrets = secrets

	configs := Configs{}
	for k := range requiredConfigs {
		if value, ok := p.Configs[k]; ok {
			configs[k] = value
		}
	}
	p.Configs = configs
}

type DependencyOption int

const (
	IncludeDependencies = iota
	IncludeDependents
	IgnoreDependencies
)

// ForServices restrict the project model to selected services and dependencies
func (p *Project) ForServices(names []string, options ...DependencyOption) error {
	if len(names) == 0 {
		// All services
		return nil
	}

	set := map[string]struct{}{}
	err := p.WithServices(names, func(service ServiceConfig) error {
		set[service.Name] = struct{}{}
		return nil
	}, options...)
	if err != nil {
		return err
	}

	// Disable all services which are not explicit target or dependencies
	var enabled Services
	for _, s := range p.Services {
		if _, ok := set[s.Name]; ok {
			for _, option := range options {
				if option == IgnoreDependencies {
					// remove all dependencies but those implied by explicitly selected services
					dependencies := s.DependsOn
					for d := range dependencies {
						if _, ok := set[d]; !ok {
							delete(dependencies, d)
						}
					}
					s.DependsOn = dependencies
				}
			}
			enabled = append(enabled, s)
		} else {
			p.DisableService(s)
		}
	}
	p.Services = enabled
	return nil
}

func (p *Project) DisableService(service ServiceConfig) {
	// We should remove all dependencies which reference the disabled service
	for i, s := range p.Services {
		if _, ok := s.DependsOn[service.Name]; ok {
			delete(s.DependsOn, service.Name)
			p.Services[i] = s
		}
	}
	p.DisabledServices = append(p.DisabledServices, service)
}

// ResolveImages updates services images to include digest computed by a resolver function
func (p *Project) ResolveImages(resolver func(named reference.Named) (godigest.Digest, error)) error {
	eg := errgroup.Group{}
	for i, s := range p.Services {
		idx := i
		service := s

		if service.Image == "" {
			continue
		}
		eg.Go(func() error {
			named, err := reference.ParseDockerRef(service.Image)
			if err != nil {
				return err
			}

			if _, ok := named.(reference.Canonical); !ok {
				// image is named but not digested reference
				digest, err := resolver(named)
				if err != nil {
					return err
				}
				named, err = reference.WithDigest(named, digest)
				if err != nil {
					return err
				}
			}

			service.Image = named.String()
			p.Services[idx] = service
			return nil
		})
	}
	return eg.Wait()
}

// MarshalYAML marshal Project into a yaml tree
func (p *Project) MarshalYAML() ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	encoder := yaml.NewEncoder(buf)
	encoder.SetIndent(2)
	// encoder.CompactSeqIndent() FIXME https://github.com/go-yaml/yaml/pull/753
	err := encoder.Encode(p)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MarshalJSON makes Config implement json.Marshaler
func (p *Project) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"name":     p.Name,
		"services": p.Services,
	}

	if len(p.Networks) > 0 {
		m["networks"] = p.Networks
	}
	if len(p.Volumes) > 0 {
		m["volumes"] = p.Volumes
	}
	if len(p.Secrets) > 0 {
		m["secrets"] = p.Secrets
	}
	if len(p.Configs) > 0 {
		m["configs"] = p.Configs
	}
	for k, v := range p.Extensions {
		m[k] = v
	}
	return json.Marshal(m)
}

// ResolveServicesEnvironment parse env_files set for services to resolve the actual environment map for services
func (p Project) ResolveServicesEnvironment(discardEnvFiles bool) error {
	for i, service := range p.Services {
		service.Environment = service.Environment.Resolve(p.Environment.Resolve)

		environment := MappingWithEquals{}
		// resolve variables based on other files we already parsed, + project's environment
		var resolve dotenv.LookupFn = func(s string) (string, bool) {
			v, ok := environment[s]
			if ok && v != nil {
				return *v, ok
			}
			return p.Environment.Resolve(s)
		}

		for _, envFile := range service.EnvFile {
			b, err := os.ReadFile(envFile)
			if err != nil {
				return errors.Wrapf(err, "Failed to load %s", envFile)
			}

			fileVars, err := dotenv.ParseWithLookup(bytes.NewBuffer(b), resolve)
			if err != nil {
				return errors.Wrapf(err, "failed to read %s", envFile)
			}
			environment.OverrideBy(Mapping(fileVars).ToMappingWithEquals())
		}

		service.Environment = environment.OverrideBy(service.Environment)

		if discardEnvFiles {
			service.EnvFile = nil
		}
		p.Services[i] = service
	}
	return nil
}
