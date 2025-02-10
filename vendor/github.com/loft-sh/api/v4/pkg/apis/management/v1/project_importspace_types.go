package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectImportSpace holds project space import information
// +subresource-request
type ProjectImportSpace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// SourceSpace is the space to import into this project
	SourceSpace ProjectImportSpaceSource `json:"sourceSpace"`
}

type ProjectImportSpaceSource struct {
	// Name of the space to import
	Name string `json:"name,omitempty"`
	// Cluster name of the cluster the space is running on
	Cluster string `json:"cluster,omitempty"`
	// ImportName is an optional name to use as the spaceinstance name, if not provided the space
	// name will be used
	// +optional
	ImportName string `json:"importName,omitempty"`
}
