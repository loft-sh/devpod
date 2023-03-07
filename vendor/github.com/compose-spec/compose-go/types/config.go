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
	"encoding/json"
	"runtime"
	"strings"

	"github.com/mitchellh/mapstructure"
)

var (
	// isCaseInsensitiveEnvVars is true on platforms where environment variable names are treated case-insensitively.
	isCaseInsensitiveEnvVars = (runtime.GOOS == "windows")
)

// ConfigDetails are the details about a group of ConfigFiles
type ConfigDetails struct {
	Version     string
	WorkingDir  string
	ConfigFiles []ConfigFile
	Environment map[string]string
}

// LookupEnv provides a lookup function for environment variables
func (cd ConfigDetails) LookupEnv(key string) (string, bool) {
	v, ok := cd.Environment[key]
	if !isCaseInsensitiveEnvVars || ok {
		return v, ok
	}
	// variable names must be treated case-insensitively on some platforms (that is, Windows).
	// Resolves in this way:
	// * Return the value if its name matches with the passed name case-sensitively.
	// * Otherwise, return the value if its lower-cased name matches lower-cased passed name.
	//     * The value is indefinite if multiple variables match.
	lowerKey := strings.ToLower(key)
	for k, v := range cd.Environment {
		if strings.ToLower(k) == lowerKey {
			return v, true
		}
	}
	return "", false
}

// ConfigFile is a filename and the contents of the file as a Dict
type ConfigFile struct {
	// Filename is the name of the yaml configuration file
	Filename string
	// Content is the raw yaml content. Will be loaded from Filename if not set
	Content []byte
	// Config if the yaml tree for this config file. Will be parsed from Content if not set
	Config map[string]interface{}
}

// Config is a full compose file configuration and model
type Config struct {
	Filename   string     `yaml:"-" json:"-"`
	Name       string     `yaml:",omitempty" json:"name,omitempty"`
	Services   Services   `json:"services"`
	Networks   Networks   `yaml:",omitempty" json:"networks,omitempty"`
	Volumes    Volumes    `yaml:",omitempty" json:"volumes,omitempty"`
	Secrets    Secrets    `yaml:",omitempty" json:"secrets,omitempty"`
	Configs    Configs    `yaml:",omitempty" json:"configs,omitempty"`
	Extensions Extensions `yaml:",inline" json:"-"`
}

// Volumes is a map of VolumeConfig
type Volumes map[string]VolumeConfig

// Networks is a map of NetworkConfig
type Networks map[string]NetworkConfig

// Secrets is a map of SecretConfig
type Secrets map[string]SecretConfig

// Configs is a map of ConfigObjConfig
type Configs map[string]ConfigObjConfig

// Extensions is a map of custom extension
type Extensions map[string]interface{}

// MarshalJSON makes Config implement json.Marshaler
func (c Config) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"services": c.Services,
	}

	if len(c.Networks) > 0 {
		m["networks"] = c.Networks
	}
	if len(c.Volumes) > 0 {
		m["volumes"] = c.Volumes
	}
	if len(c.Secrets) > 0 {
		m["secrets"] = c.Secrets
	}
	if len(c.Configs) > 0 {
		m["configs"] = c.Configs
	}
	for k, v := range c.Extensions {
		m[k] = v
	}
	return json.Marshal(m)
}

func (e Extensions) Get(name string, target interface{}) (bool, error) {
	if v, ok := e[name]; ok {
		err := mapstructure.Decode(v, target)
		return true, err
	}
	return false, nil
}
