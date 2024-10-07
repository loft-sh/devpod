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
	"fmt"
)

type SSHKey struct {
	ID   string `yaml:"id,omitempty" json:"id,omitempty"`
	Path string `path:"path,omitempty" json:"path,omitempty"`
}

// SSHConfig is a mapping type for SSH build config
type SSHConfig []SSHKey

func (s SSHConfig) Get(id string) (string, error) {
	for _, sshKey := range s {
		if sshKey.ID == id {
			return sshKey.Path, nil
		}
	}
	return "", fmt.Errorf("ID %s not found in SSH keys", id)
}

// MarshalYAML makes SSHKey implement yaml.Marshaller
func (s SSHKey) MarshalYAML() (interface{}, error) {
	if s.Path == "" {
		return s.ID, nil
	}
	return fmt.Sprintf("%s: %s", s.ID, s.Path), nil
}

// MarshalJSON makes SSHKey implement json.Marshaller
func (s SSHKey) MarshalJSON() ([]byte, error) {
	if s.Path == "" {
		return []byte(fmt.Sprintf(`%q`, s.ID)), nil
	}
	return []byte(fmt.Sprintf(`%q: %s`, s.ID, s.Path)), nil
}

func (s *SSHConfig) DecodeMapstructure(value interface{}) error {
	v, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid ssh config type %T", value)
	}
	result := make(SSHConfig, len(v))
	i := 0
	for id, path := range v {
		key := SSHKey{ID: id}
		if path != nil {
			key.Path = fmt.Sprint(path)
		}
		result[i] = key
		i++
	}
	*s = result
	return nil
}
