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
)

type EnvFile struct {
	Path     string `yaml:"path,omitempty" json:"path,omitempty"`
	Required bool   `yaml:"required" json:"required"`
}

// MarshalYAML makes EnvFile implement yaml.Marshaler
func (e EnvFile) MarshalYAML() (interface{}, error) {
	if e.Required {
		return e.Path, nil
	}
	return map[string]any{
		"path":     e.Path,
		"required": e.Required,
	}, nil
}

// MarshalJSON makes EnvFile implement json.Marshaler
func (e *EnvFile) MarshalJSON() ([]byte, error) {
	if e.Required {
		return json.Marshal(e.Path)
	}
	// Pass as a value to avoid re-entering this method and use the default implementation
	return json.Marshal(*e)
}
