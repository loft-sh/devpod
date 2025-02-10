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
	"strconv"
	"strings"
)

type DeviceRequest struct {
	Capabilities []string    `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	Driver       string      `yaml:"driver,omitempty" json:"driver,omitempty"`
	Count        DeviceCount `yaml:"count,omitempty" json:"count,omitempty"`
	IDs          []string    `yaml:"device_ids,omitempty" json:"device_ids,omitempty"`
}

type DeviceCount int64

func (c *DeviceCount) DecodeMapstructure(value interface{}) error {
	switch v := value.(type) {
	case int:
		*c = DeviceCount(v)
	case string:
		if strings.ToLower(v) == "all" {
			*c = -1
			return nil
		}
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid value %q, the only value allowed is 'all' or a number", v)
		}
		*c = DeviceCount(i)
	default:
		return fmt.Errorf("invalid type %T for device count", v)
	}
	return nil
}
