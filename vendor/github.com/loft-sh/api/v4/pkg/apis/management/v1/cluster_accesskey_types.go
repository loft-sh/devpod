package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterAccessKey holds the access key for the cluster
// +subresource-request
type ClusterAccessKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// AccessKey is the access key used by the agent
	// +optional
	AccessKey string `json:"accessKey,omitempty"`

	// LoftHost is the loft host used by the agent
	// +optional
	LoftHost string `json:"loftHost,omitempty"`

	// Insecure signals if the loft host is insecure
	// +optional
	Insecure bool `json:"insecure,omitempty"`

	// CaCert is an optional ca cert to use for the loft host connection
	// +optional
	CaCert string `json:"caCert,omitempty"`
}
