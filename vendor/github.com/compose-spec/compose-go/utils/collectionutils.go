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

import "golang.org/x/exp/slices"

func MapKeys[T comparable, U any](theMap map[T]U) []T {
	var result []T
	for key := range theMap {
		result = append(result, key)
	}
	return result
}

func MapsAppend[T comparable, U any](target map[T]U, source map[T]U) map[T]U {
	if target == nil {
		return source
	}
	if source == nil {
		return target
	}
	for key, value := range source {
		if _, ok := target[key]; !ok {
			target[key] = value
		}
	}
	return target
}

func ArrayContains[T comparable](source []T, toCheck []T) bool {
	for _, value := range toCheck {
		if !slices.Contains(source, value) {
			return false
		}
	}
	return true
}
