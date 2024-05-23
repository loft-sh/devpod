package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectMigrateSpaceInstance holds project spaceinstance migrate information
// +subresource-request
type ProjectMigrateSpaceInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// SourceSpaceInstance is the spaceinstance to migrate into this project
	SourceSpaceInstance ProjectMigrateSpaceInstanceSource `json:"sourceSpaceInstance"`
}

type ProjectMigrateSpaceInstanceSource struct {
	// Name of the spaceinstance to migrate
	Name string `json:"name,omitempty"`
	// Namespace of the spaceinstance to migrate
	Namespace string `json:"namespace,omitempty"`
}
