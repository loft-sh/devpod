package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LEGACY: Please use access keys + direct cluster endpoint

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DirectClusterEndpointToken holds the object information
// +k8s:openapi-gen=true
// +resource:path=directclusterendpointtokens,rest=DirectClusterEndpointTokenREST
type DirectClusterEndpointToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DirectClusterEndpointTokenSpec   `json:"spec,omitempty"`
	Status DirectClusterEndpointTokenStatus `json:"status,omitempty"`
}

// DirectClusterEndpointTokenSpec holds the object specification
type DirectClusterEndpointTokenSpec struct {
	// The time to life for this access token in seconds
	// +optional
	TTL int64 `json:"ttl,omitempty"`

	// Scope is the optional scope of the direct cluster endpoint
	// +optional
	Scope *storagev1.AccessKeyScope `json:"scope,omitempty"`
}

// DirectClusterEndpointTokenStatus holds the object status
type DirectClusterEndpointTokenStatus struct {
	// +optional
	Token string `json:"token,omitempty"`
}
