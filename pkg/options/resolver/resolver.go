package resolver

import (
	"context"
	"regexp"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/graph"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/log"
)

const (
	rootID = "root"
)

var variableExpression = regexp.MustCompile(`(?m)\$\{?([A-Z0-9_]+)(:(-|\+)([^\}]+))?\}?`)

func ResolveDefaultValue(val string, resolvedOptions map[string]string) string {
	return variableExpression.ReplaceAllStringFunc(val, func(s string) string {
		submatch := variableExpression.FindStringSubmatch(s)
		optionVal, ok := resolvedOptions[submatch[1]]
		if ok {
			return optionVal
		}

		return s
	})
}

func ResolveDefaultValues(vals map[string]string, resolvedOptions map[string]string) map[string]string {
	ret := make(map[string]string)
	for k, v := range vals {
		resolvedValue := ResolveDefaultValue(v, resolvedOptions)
		if resolvedValue == "" {
			continue
		}

		ret[k] = resolvedValue
	}
	return ret
}

type Resolver struct {
	// user provided
	userOptions map[string]string
	// extra values
	extraValues map[string]string

	// internal
	graph *graph.Graph[*types.Option]
	log   log.Logger

	// options
	resolveLocal      bool
	resolveGlobal     bool
	resolveSubOptions bool
	skipRequired      bool
}

type Option func(r *Resolver)

func New(userOptions map[string]string, extraValues map[string]string, logger log.Logger, opts ...Option) *Resolver {
	if userOptions == nil {
		userOptions = map[string]string{}
	}

	resolver := &Resolver{
		userOptions: userOptions,
		extraValues: extraValues,
		log:         logger,
	}
	for _, o := range opts {
		o(resolver)
	}

	return resolver
}

func WithResolveLocal() Option {
	return func(r *Resolver) {
		r.resolveLocal = true
	}
}

func WithResolveGlobal() Option {
	return func(r *Resolver) {
		r.resolveGlobal = true
	}
}

func WithResolveSubOptions() Option {
	return func(r *Resolver) {
		r.resolveSubOptions = true
	}
}

func WithSkipRequired(skip bool) Option {
	return func(r *Resolver) {
		r.skipRequired = skip
	}
}

func (r *Resolver) Resolve(
	ctx context.Context,
	dynamicDefinitions map[string]*types.Option,
	optionDefinitions map[string]*types.Option,
	optionValues map[string]config.OptionValue,
) (map[string]config.OptionValue, config.OptionDefinitions, error) {
	if dynamicDefinitions == nil {
		dynamicDefinitions = map[string]*types.Option{}
	}
	if optionDefinitions == nil {
		optionDefinitions = map[string]*types.Option{}
	}
	mergedOptionDefinitions := mergeMaps(dynamicDefinitions, optionDefinitions)

	// create a new graph, which we resolve from top to bottom, where a child represents an
	// option that is dependent on the parent. Parents will be resolved first.
	r.graph = graph.NewGraphOf(graph.NewNode[*types.Option](rootID, nil), "provider option")
	err := addOptionsToGraph(r.graph, mergedOptionDefinitions, optionValues)
	if err != nil {
		return nil, nil, err
	}

	// resolve options
	resolvedOptions, err := r.resolveOptions(ctx, optionValues)
	if err != nil {
		return nil, nil, err
	}

	// find out new dynamic options
	newDynamicDefinitions := config.OptionDefinitions{}
	for k, node := range r.graph.Nodes {
		if k == rootID || optionDefinitions[k] != nil {
			continue
		}

		// check if someone has the option as children
		for _, v := range resolvedOptions {
			if contains(v.Children, k) {
				newDynamicDefinitions[k] = node.Data
				break
			}
		}
	}

	// remove options that are not there anymore
	for k := range resolvedOptions {
		if newDynamicDefinitions[k] == nil && optionDefinitions[k] == nil {
			delete(resolvedOptions, k)
		}
	}

	// print unused user values
	if !r.skipRequired {
		printUnusedUserValues(r.userOptions, mergeMaps(optionDefinitions, newDynamicDefinitions), r.log)
	}

	return resolvedOptions, newDynamicDefinitions, nil
}
