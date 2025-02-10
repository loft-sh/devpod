/*
Copyright 2019 The Kubernetes Authors.
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

import "k8s.io/apiserver/pkg/registry/rest"

var _ rest.ShortNamesProvider = &StorageWrapperWithShortcuts{}
var _ rest.CategoriesProvider = &StorageWrapperWithShortcuts{}

type StorageWrapperWithShortcuts struct {
	*StorageWrapper
	shortNames []string
	categories []string
}

func (b *StorageWrapperWithShortcuts) ShortNames() []string {
	// TODO(yue9944882): prevent shortname conflict or the client-side rest-mapping will crush
	return b.shortNames
}

func (b *StorageWrapperWithShortcuts) Categories() []string {
	// all the aggregated resource are considered in the "aggregation" category
	return b.categories
}
