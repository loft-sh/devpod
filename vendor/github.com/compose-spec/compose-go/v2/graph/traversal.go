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
	"sync"

	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
)

// CollectorFn executes on each graph vertex based on visit order and return associated value
type CollectorFn[S any, T any] func(context.Context, string, S) (T, error)

// VisitorFn executes on each graph nodes based on visit order
type VisitorFn[S any] func(context.Context, string, S) error

type traversal[S any, T any] struct {
	*Options
	visitor CollectorFn[S, T]

	mu      sync.Mutex
	status  map[string]int
	results map[string]T
}

type Options struct {
	// inverse reverse the traversal direction
	inverse bool
	// maxConcurrency limit the concurrent execution of visitorFn while walking the graph
	maxConcurrency int
	// after marks a set of node as starting points walking the graph
	after []string
}

const (
	vertexEntered = iota
	vertexVisited
)

func newTraversal[S, T any](fn CollectorFn[S, T]) *traversal[S, T] {
	return &traversal[S, T]{
		Options: &Options{},
		status:  map[string]int{},
		results: map[string]T{},
		visitor: fn,
	}
}

// WithMaxConcurrency configure traversal to limit concurrency walking graph nodes
func WithMaxConcurrency(max int) func(*Options) {
	return func(o *Options) {
		o.maxConcurrency = max
	}
}

// InReverseOrder configure traversal to walk the graph in reverse dependency order
func InReverseOrder(o *Options) {
	o.inverse = true
}

// WithRootNodesAndDown creates a graphTraversal to start from selected nodes
func WithRootNodesAndDown(nodes []string) func(*Options) {
	return func(o *Options) {
		o.after = nodes
	}
}

func walk[S, T any](ctx context.Context, g *graph[S], t *traversal[S, T]) error {
	expect := len(g.vertices)
	if expect == 0 {
		return nil
	}
	// nodeCh need to allow n=expect writers while reader goroutine could have returned after ctx.Done
	nodeCh := make(chan *vertex[S], expect)
	defer close(nodeCh)

	eg, ctx := errgroup.WithContext(ctx)
	if t.maxConcurrency > 0 {
		eg.SetLimit(t.maxConcurrency + 1)
	}

	eg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case node := <-nodeCh:
				expect--
				if expect == 0 {
					return nil
				}

				for _, adj := range t.adjacentNodes(node) {
					t.visit(ctx, eg, adj, nodeCh)
				}
			}
		}
	})

	// select nodes to start walking the graph based on traversal.direction
	for _, node := range t.extremityNodes(g) {
		t.visit(ctx, eg, node, nodeCh)
	}

	return eg.Wait()
}

func (t *traversal[S, T]) visit(ctx context.Context, eg *errgroup.Group, node *vertex[S], nodeCh chan *vertex[S]) {
	if !t.ready(node) {
		// don't visit this service yet as dependencies haven't been visited
		return
	}
	if !t.enter(node) {
		// another worker already acquired this node
		return
	}
	eg.Go(func() error {
		var (
			err    error
			result T
		)
		if !t.skip(node) {
			result, err = t.visitor(ctx, node.key, *node.service)
		}
		t.done(node, result)
		nodeCh <- node
		return err
	})
}

func (t *traversal[S, T]) extremityNodes(g *graph[S]) []*vertex[S] {
	if t.inverse {
		return g.roots()
	}
	return g.leaves()
}

func (t *traversal[S, T]) adjacentNodes(v *vertex[S]) map[string]*vertex[S] {
	if t.inverse {
		return v.children
	}
	return v.parents
}

func (t *traversal[S, T]) ready(v *vertex[S]) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	depends := v.children
	if t.inverse {
		depends = v.parents
	}
	for name := range depends {
		if t.status[name] != vertexVisited {
			return false
		}
	}
	return true
}

func (t *traversal[S, T]) enter(v *vertex[S]) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.status[v.key]; ok {
		return false
	}
	t.status[v.key] = vertexEntered
	return true
}

func (t *traversal[S, T]) done(v *vertex[S], result T) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status[v.key] = vertexVisited
	t.results[v.key] = result
}

func (t *traversal[S, T]) skip(node *vertex[S]) bool {
	if len(t.after) == 0 {
		return false
	}
	if slices.Contains(t.after, node.key) {
		return false
	}

	// is none of our starting node is a descendent, skip visit
	ancestors := node.descendents()
	for _, name := range t.after {
		if slices.Contains(ancestors, name) {
			return false
		}
	}
	return true
}
