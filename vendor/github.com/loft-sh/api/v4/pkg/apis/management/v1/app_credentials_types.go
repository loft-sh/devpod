package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type AppCredentials struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// ProjectSecretRefs holds the resolved secret values for the project secret refs.
	// +optional
	ProjectSecretRefs map[string]string `json:"projectSecretRefs,omitempty"`
}
