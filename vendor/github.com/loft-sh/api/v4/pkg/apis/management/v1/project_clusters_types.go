package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type ProjectClusters struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Clusters holds all the allowed clusters
	Clusters []Cluster `json:"clusters,omitempty"`

	// Runners holds all the allowed runners
	Runners []Runner `json:"runners,omitempty"`
}
