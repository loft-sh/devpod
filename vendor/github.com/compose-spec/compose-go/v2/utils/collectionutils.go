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
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func MapKeys[T constraints.Ordered, U any](theMap map[T]U) []T {
	result := maps.Keys(theMap)
	slices.Sort(result)
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

func RemoveDuplicates[T comparable](slice []T) []T {
	// Create a map to store unique elements
	seen := make(map[T]bool)
	result := []T{}

	// Loop through the slice, adding elements to the map if they haven't been seen before
	for _, val := range slice {
		if _, ok := seen[val]; !ok {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
}
