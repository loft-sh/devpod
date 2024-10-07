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
	"context"
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
)

// InDependencyOrder walk the service graph an invoke VisitorFn in respect to dependency order
func InDependencyOrder(ctx context.Context, project *types.Project, fn VisitorFn[types.ServiceConfig], options ...func(*Options)) error {
	_, err := CollectInDependencyOrder[any](ctx, project, func(ctx context.Context, s string, config types.ServiceConfig) (any, error) {
		return nil, fn(ctx, s, config)
	}, options...)
	return err
}

// CollectInDependencyOrder walk the service graph an invoke CollectorFn in respect to dependency order, then return result for each call
func CollectInDependencyOrder[T any](ctx context.Context, project *types.Project, fn CollectorFn[types.ServiceConfig, T], options ...func(*Options)) (map[string]T, error) {
	graph, err := newGraph(project)
	if err != nil {
		return nil, err
	}
	t := newTraversal(fn)
	for _, option := range options {
		option(t.Options)
	}
	err = walk(ctx, graph, t)
	return t.results, err
}

// newGraph creates a service graph from project
func newGraph(project *types.Project) (*graph[types.ServiceConfig], error) {
	g := &graph[types.ServiceConfig]{
		vertices: map[string]*vertex[types.ServiceConfig]{},
	}

	for name, s := range project.Services {
		g.addVertex(name, s)
	}

	for name, s := range project.Services {
		src := g.vertices[name]
		for dep, condition := range s.DependsOn {
			dest, ok := g.vertices[dep]
			if !ok {
				if condition.Required {
					if ds, exists := project.DisabledServices[dep]; exists {
						return nil, fmt.Errorf("service %q is required by %q but is disabled. Can be enabled by profiles %s", dep, name, ds.Profiles)
					}
					return nil, fmt.Errorf("service %q depends on unknown service %q", name, dep)
				}
				delete(s.DependsOn, name)
				project.Services[name] = s
				continue
			}
			src.children[dep] = dest
			dest.parents[name] = src
		}
	}

	err := g.checkCycle()
	return g, err
}
