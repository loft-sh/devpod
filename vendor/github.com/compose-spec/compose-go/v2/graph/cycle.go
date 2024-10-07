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

package graph

import (
	"fmt"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/compose-spec/compose-go/v2/utils"
	"golang.org/x/exp/slices"
)

// CheckCycle analyze project's depends_on relation and report an error on cycle detection
func CheckCycle(project *types.Project) error {
	g, err := newGraph(project)
	if err != nil {
		return err
	}
	return g.checkCycle()
}

func (g *graph[T]) checkCycle() error {
	// iterate on vertices in a name-order to render a predicable error message
	// this is required by tests and enforce command reproducibility by user, which otherwise could be confusing
	names := utils.MapKeys(g.vertices)
	for _, name := range names {
		err := searchCycle([]string{name}, g.vertices[name])
		if err != nil {
			return err
		}
	}
	return nil
}

func searchCycle[T any](path []string, v *vertex[T]) error {
	names := utils.MapKeys(v.children)
	for _, name := range names {
		if i := slices.Index(path, name); i >= 0 {
			return fmt.Errorf("dependency cycle detected: %s -> %s", strings.Join(path[i:], " -> "), name)
		}
		ch := v.children[name]
		err := searchCycle(append(path, name), ch)
		if err != nil {
			return err
		}
	}
	return nil
}
