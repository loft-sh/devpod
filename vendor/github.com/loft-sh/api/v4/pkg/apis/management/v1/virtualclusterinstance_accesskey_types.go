package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterAccessKey holds the access key for the virtual cluster
// +subresource-request
type VirtualClusterAccessKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// AccessKey is the access key used by the agent
	// +optional
	AccessKey string `json:"accessKey,omitempty"`
}
