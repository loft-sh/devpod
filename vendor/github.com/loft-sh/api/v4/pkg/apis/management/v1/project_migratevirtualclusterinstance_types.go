package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectMigrateVirtualClusterInstance holds project vclusterinstance migrate information
// +subresource-request
type ProjectMigrateVirtualClusterInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// SourceVirtualClusterInstance is the virtual cluster instance to migrate into this project
	SourceVirtualClusterInstance ProjectMigrateVirtualClusterInstanceSource `json:"sourceVirtualClusterInstance"`
}

type ProjectMigrateVirtualClusterInstanceSource struct {
	// Name of the virtual cluster instance to migrate
	Name string `json:"name,omitempty"`
	// Namespace of the virtual cluster instance to migrate
	Namespace string `json:"namespace,omitempty"`
}
