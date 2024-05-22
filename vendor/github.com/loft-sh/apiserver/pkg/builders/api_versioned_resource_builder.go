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
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
)

type NewRESTFunc func(getter generic.RESTOptionsGetter) rest.Storage

//
// Versioned Kind Builder builds a versioned resource using unversioned strategy
//

// NewApiResource returns a new versionedResourceBuilder for registering endpoints for
// resources that are persisted to storage.
// strategy - unversionedBuilder from calling NewUnversionedXXX()
// new - function for creating new empty VERSIONED instances - e.g. func() runtime.Object { return &Deployment{} }
// newList - function for creating an empty list of VERSIONED instances - e.g. func() runtime.Object { return &DeploymentList{} }
// storeBuilder - builder for creating the store
func NewApiResource(
	unversionedBuilder UnversionedResourceBuilder,
	new, newList func() runtime.Object,
	storeBuilder StorageBuilder) *versionedResourceBuilder {

	return &versionedResourceBuilder{
		unversionedBuilder, new, newList, storeBuilder, nil, nil,
	}
}

// NewApiResourceWithStorage returns a new versionedResourceBuilder for registering endpoints that
// does not require standard storage (e.g. subresources reuses the storage for the parent resource).
// strategy - unversionedBuilder from calling NewUnversionedXXX()
// new - function for creating new empty VERSIONED instances - e.g. func() runtime.Object { return &Deployment{} }
// storage - storage for manipulating the resource
func NewApiResourceWithStorage(
	unversionedBuilder UnversionedResourceBuilder,
	new, newList func() runtime.Object,
	RESTFunc NewRESTFunc) *versionedResourceBuilder {
	v := &versionedResourceBuilder{
		unversionedBuilder, new, newList, nil, RESTFunc, nil,
	}
	if new == nil {
		panic(fmt.Errorf("Cannot call NewApiResourceWithStorage with nil new function."))
	}
	if RESTFunc == nil {
		panic(fmt.Errorf("Cannot call NewApiResourceWithStorage with nil RESTFunc function."))
	}
	return v
}

type versionedResourceBuilder struct {
	Unversioned UnversionedResourceBuilder

	// NewFunc returns an empty unversioned instance of a resource
	NewFunc func() runtime.Object

	// NewListFunc returns and empty unversioned instance of a resource List
	NewListFunc func() runtime.Object

	// StorageBuilder is used to modify the default storage, mutually exclusive with RESTFunc
	StorageBuilder StorageBuilder

	// RESTFunc returns a rest.Storage implementation, mutually exclusive with StorageBuilder
	RESTFunc NewRESTFunc

	Storage rest.StandardStorage
}

func (b *versionedResourceBuilder) New() runtime.Object {
	if b.NewFunc == nil {
		return nil
	}
	return b.NewFunc()
}

func (b *versionedResourceBuilder) NewList() runtime.Object {
	if b.NewListFunc == nil {
		return nil
	}
	return b.NewListFunc()
}

type StorageWrapper struct {
	registry.Store
}

func (s StorageWrapper) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	return s.Store.Create(ctx, obj, createValidation, options)
}

func (b *versionedResourceBuilder) Build(
	group string,
	optionsGetter generic.RESTOptionsGetter) rest.StandardStorage {

	// Set a default strategy
	store := &StorageWrapper{
		registry.Store{
			NewFunc:                  b.Unversioned.New,     // Use the unversioned type
			NewListFunc:              b.Unversioned.NewList, // Use the unversioned type
			DefaultQualifiedResource: b.getGroupResource(group),
		},
	}
	b.Storage = store

	// the store-with-shortcuts will only be used if there're valid shortnames
	storeWithShortcuts := &StorageWrapperWithShortcuts{
		StorageWrapper: store,
	}

	wantsShortcuts := len(b.Unversioned.GetShortNames()) > 0
	if wantsShortcuts {
		// plants shortnames and an opt-out category into the storage
		storeWithShortcuts.shortNames = b.Unversioned.GetShortNames()
		storeWithShortcuts.categories = b.Unversioned.GetCategories()
	}

	// Use default, requires
	options := &generic.StoreOptions{RESTOptions: optionsGetter}

	if b.StorageBuilder != nil {
		// Allow overriding the storage defaults
		b.StorageBuilder.Build(b.StorageBuilder, storeWithShortcuts.StorageWrapper, options)
	}

	if err := storeWithShortcuts.CompleteWithOptions(options); err != nil {
		panic(err) // TODO: Propagate error up
	}
	if wantsShortcuts {
		b.Storage = storeWithShortcuts
	}
	return b.Storage
}

func (b *versionedResourceBuilder) GetStandardStorage() rest.StandardStorage {
	return b.Storage
}

// getGroupResource returns the GroupResource for this Resource and the provided Group
// group is the group the resource belongs to
func (b *versionedResourceBuilder) getGroupResource(group string) schema.GroupResource {
	return schema.GroupResource{Group: group, Resource: b.Unversioned.GetName()}

}

// registerEndpoints registers the REST endpoints for this resource in the registry
// group is the group to register the resource under
// optionsGetter is the RESTOptionsGetter provided by a server.Config
// registry is the server.APIGroupInfo VersionedResourcesStorageMap used to register REST endpoints
func (b *versionedResourceBuilder) registerEndpoints(
	group string,
	optionsGetter generic.RESTOptionsGetter,
	registry map[string]rest.Storage) {

	// Register the endpoint
	path := b.Unversioned.GetPath()
	if len(path) > 0 {
		// Subresources appear after the resource
		path = b.Unversioned.GetName() + "/" + path
	} else {
		path = b.Unversioned.GetName()
	}

	if b.RESTFunc != nil {
		// Use the REST implementation directly.
		registry[path] = b.RESTFunc(optionsGetter)
	} else {
		// Create a new REST implementation wired to storage.
		registry[path] = b.
			Build(group, optionsGetter)
	}
}
