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
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
)

type VersionedApiBuilder struct {
	Kinds         []*versionedResourceBuilder
	GroupVersion  schema.GroupVersion
	SchemeBuilder runtime.SchemeBuilder

	// Deprecated: Only for compiliation backward-compatibility w/ 1.12+ version generators
	// removing in the future
	SchemaBuilder runtime.SchemeBuilder
}

func NewApiVersion(group, version string) *VersionedApiBuilder {
	b := &VersionedApiBuilder{
		GroupVersion: schema.GroupVersion{Group: group, Version: version},
	}
	return b
}

// WithResources adds new resource types and subresources to the API versions
// resourceBuilders is a list of *versionedResourceBuilder
func (s *VersionedApiBuilder) WithResources(resourceBuilders ...*versionedResourceBuilder) *VersionedApiBuilder {
	s.Kinds = append(s.Kinds, resourceBuilders...)
	return s
}

// registerEndpoints registers the REST endpoints for all resources in this API group version
// group is the group to register the resources under
// optionsGetter is the RESTOptionsGetter provided by a server.Config
// registry is the server.APIGroupInfo VersionedResourcesStorageMap used to register REST endpoints
func (s *VersionedApiBuilder) registerEndpoints(
	optionsGetter generic.RESTOptionsGetter,
	registry map[string]map[string]rest.Storage) {

	// Register the endpoints for each kind
	for _, k := range s.Kinds {
		if _, found := registry[s.GroupVersion.Version]; !found {
			// Initialize the version if missing
			registry[s.GroupVersion.Version] = map[string]rest.Storage{}
		}
		// Register each of the endpoints in this version
		k.registerEndpoints(s.GroupVersion.Group, optionsGetter, registry[s.GroupVersion.Version])
	}
}

type UnVersionedApiBuilder struct {
	Kinds         []UnversionedResourceBuilder
	GroupVersion  schema.GroupVersion
	SchemeBuilder runtime.SchemeBuilder
}

func NewApiGroup(group string) *UnVersionedApiBuilder {
	b := &UnVersionedApiBuilder{
		GroupVersion: schema.GroupVersion{Group: group, Version: runtime.APIVersionInternal},
	}
	//b.SchemaBuilder = runtime.NewSchemeBuilder(b.registerTypes)
	return b
}

func (s *UnVersionedApiBuilder) WithKinds(kinds ...UnversionedResourceBuilder) *UnVersionedApiBuilder {
	s.Kinds = append(s.Kinds, kinds...)
	return s
}

func (s *UnVersionedApiBuilder) registerTypes(scheme *runtime.Scheme) error {
	// Make sure we only each type once
	typesToRegister := map[reflect.Type]runtime.Object{}
	for _, k := range s.Kinds {
		// RegisterTypes type
		if t := k.New(); t != nil {
			// Register the unversioned type
			typesToRegister[reflect.TypeOf(t)] = t
		}

		// RegisterTypes list type if it exists
		if i, ok := k.(WithList); ok && i.NewList() != nil {
			// Register the unversioned type
			l := i.NewList()
			typesToRegister[reflect.TypeOf(l)] = l
		}
	}
	for _, t := range typesToRegister {
		scheme.AddKnownTypes(s.GroupVersion, t)
	}

	//fmt.Printf("Registering for group %v\n", s.GroupVersion)
	//metav1.AddToGroupVersion(scheme, s.GroupVersion)
	return nil
}
