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

// graph represents project as service dependencies
type graph[T any] struct {
	vertices map[string]*vertex[T]
}

// vertex represents a service in the dependencies structure
type vertex[T any] struct {
	key      string
	service  *T
	children map[string]*vertex[T]
	parents  map[string]*vertex[T]
}

func (g *graph[T]) addVertex(name string, service T) {
	g.vertices[name] = &vertex[T]{
		key:      name,
		service:  &service,
		parents:  map[string]*vertex[T]{},
		children: map[string]*vertex[T]{},
	}
}

func (g *graph[T]) addEdge(src, dest string) {
	g.vertices[src].children[dest] = g.vertices[dest]
	g.vertices[dest].parents[src] = g.vertices[src]
}

func (g *graph[T]) roots() []*vertex[T] {
	var res []*vertex[T]
	for _, v := range g.vertices {
		if len(v.parents) == 0 {
			res = append(res, v)
		}
	}
	return res
}

func (g *graph[T]) leaves() []*vertex[T] {
	var res []*vertex[T]
	for _, v := range g.vertices {
		if len(v.children) == 0 {
			res = append(res, v)
		}
	}

	return res
}

// descendents return all descendents for a vertex, might contain duplicates
func (v *vertex[T]) descendents() []string {
	var vx []string
	for _, n := range v.children {
		vx = append(vx, n.key)
		vx = append(vx, n.descendents()...)
	}
	return vx
}
