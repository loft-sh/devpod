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
	"strings"
)

// Labels is a mapping type for labels
type Labels map[string]string

func (l Labels) Add(key, value string) Labels {
	if l == nil {
		l = Labels{}
	}
	l[key] = value
	return l
}

func (l Labels) AsList() []string {
	s := make([]string, len(l))
	i := 0
	for k, v := range l {
		s[i] = fmt.Sprintf("%s=%s", k, v)
		i++
	}
	return s
}

// label value can be a string | number | boolean | null (empty)
func labelValue(e interface{}) string {
	if e == nil {
		return ""
	}
	switch v := e.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

func (l *Labels) DecodeMapstructure(value interface{}) error {
	switch v := value.(type) {
	case map[string]interface{}:
		labels := make(map[string]string, len(v))
		for k, e := range v {
			labels[k] = labelValue(e)
		}
		*l = labels
	case []interface{}:
		labels := make(map[string]string, len(v))
		for _, s := range v {
			k, e, _ := strings.Cut(fmt.Sprint(s), "=")
			labels[k] = labelValue(e)
		}
		*l = labels
	default:
		return fmt.Errorf("unexpected value type %T for labels", value)
	}
	return nil
}
