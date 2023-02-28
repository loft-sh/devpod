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

package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// StringContains check if an array contains a specific value
func StringContains(array []string, needle string) bool {
	for _, val := range array {
		if val == needle {
			return true
		}
	}
	return false
}

// StringToBool converts a string to a boolean ignoring errors
func StringToBool(s string) bool {
	b, _ := strconv.ParseBool(strings.ToLower(strings.TrimSpace(s)))
	return b
}

// GetAsEqualsMap split key=value formatted strings into a key : value map
func GetAsEqualsMap(em []string) map[string]string {
	m := make(map[string]string)
	for _, v := range em {
		kv := strings.SplitN(v, "=", 2)
		m[kv[0]] = kv[1]
	}
	return m
}

// GetAsEqualsMap format a key : value map into key=value strings
func GetAsStringList(em map[string]string) []string {
	m := make([]string, 0, len(em))
	for k, v := range em {
		m = append(m, fmt.Sprintf("%s=%s", k, v))
	}
	return m
}
