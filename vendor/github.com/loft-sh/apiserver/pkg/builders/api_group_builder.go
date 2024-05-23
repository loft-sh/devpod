/*
Copyright 2017 The Kubernetes Authors.
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

package builders

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
)

type APIGroupBuilder struct {
	UnVersioned     *UnVersionedApiBuilder
	Versions        []*VersionedApiBuilder
	Name            string
	ImportPrefix    string
	RootScopedKinds []string
}

func NewApiGroupBuilder(name, prefix string) *APIGroupBuilder {
	g := &APIGroupBuilder{
		Name:         name,
		ImportPrefix: prefix,
	}
	return g
}

func (g *APIGroupBuilder) WithUnVersionedApi(unversioned *UnVersionedApiBuilder) *APIGroupBuilder {
	g.UnVersioned = unversioned
	return g
}

func (g *APIGroupBuilder) WithVersionedApis(versions ...*VersionedApiBuilder) *APIGroupBuilder {
	g.Versions = append(g.Versions, versions...)
	return g
}

func (g *APIGroupBuilder) WithRootScopedKinds(kinds ...string) *APIGroupBuilder {
	g.RootScopedKinds = append(g.RootScopedKinds, kinds...)
	return g
}

// GetVersionPreferenceOrder returns the preferred ordering of versions for this api group
func (g *APIGroupBuilder) GetVersionPreferenceOrder() []string {
	order := []string{}
	for _, v := range g.Versions {
		order = append(order, v.GroupVersion.Version)
	}
	return order
}

func (g *APIGroupBuilder) GetLegacyCodec() []schema.GroupVersion {
	versions := []schema.GroupVersion{}
	for _, v := range g.Versions {
		versions = append(versions, v.GroupVersion)
	}
	return versions
}

func (g *APIGroupBuilder) registerEndpoints(
	optionsGetter generic.RESTOptionsGetter,
	registry map[string]map[string]rest.Storage) {

	// Register the endpoints for each version
	for _, v := range g.Versions {
		v.registerEndpoints(optionsGetter, registry)
	}
}

func (g *APIGroupBuilder) registerVersionPriorities(scheme *runtime.Scheme) error {
	gvs := make([]schema.GroupVersion, len(g.Versions))
	for i, versionBuilder := range g.Versions {
		gvs[i] = versionBuilder.GroupVersion
	}
	return scheme.SetVersionPriority(gvs...)
}

// Build returns a new NewDefaultAPIGroupInfo to install into a GenericApiServer
func (g *APIGroupBuilder) Build(optionsGetter generic.RESTOptionsGetter) *genericapiserver.APIGroupInfo {

	// Build a new group
	i := genericapiserver.NewDefaultAPIGroupInfo(
		g.Name,
		Scheme,
		ParameterCodec,
		Codecs)

	g.registerEndpoints(optionsGetter, i.VersionedResourcesStorageMap)
	return &i
}

func (g *APIGroupBuilder) AddToScheme(scheme *runtime.Scheme) error {
	localSchemeBuilder := runtime.NewSchemeBuilder(g.registerVersionPriorities)
	for _, versionBuilder := range g.Versions {
		localSchemeBuilder.Register(versionBuilder.SchemeBuilder.AddToScheme)
	}
	localSchemeBuilder.Register(g.UnVersioned.SchemeBuilder.AddToScheme)
	return localSchemeBuilder.AddToScheme(scheme)
}
