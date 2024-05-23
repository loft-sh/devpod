package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RunnerAccessKey holds the access key for the runner
// +subresource-request
type RunnerAccessKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// AccessKey is the access key used by the runner
	// +optional
	AccessKey string `json:"accessKey,omitempty"`
}
