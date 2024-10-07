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
	"fmt"
	"strings"
	"time"
)

// Duration is a thin wrapper around time.Duration with improved JSON marshalling
type Duration time.Duration

func (d Duration) String() string {
	return time.Duration(d).String()
}

func (d *Duration) DecodeMapstructure(value interface{}) error {
	v, err := time.ParseDuration(fmt.Sprint(value))
	if err != nil {
		return err
	}
	*d = Duration(v)
	return nil
}

// MarshalJSON makes Duration implement json.Marshaler
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// MarshalYAML makes Duration implement yaml.Marshaler
func (d Duration) MarshalYAML() (interface{}, error) {
	return d.String(), nil
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	timeDuration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(timeDuration)
	return nil
}
