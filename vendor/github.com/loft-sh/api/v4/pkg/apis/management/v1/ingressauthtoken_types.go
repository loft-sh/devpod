package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IngressAuthToken holds the object information
// +k8s:openapi-gen=true
// +resource:path=ingressauthtokens,rest=IngressAuthTokenREST
type IngressAuthToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IngressAuthTokenSpec   `json:"spec,omitempty"`
	Status IngressAuthTokenStatus `json:"status,omitempty"`
}

// IngressAuthTokenSpec holds the object specification
type IngressAuthTokenSpec struct {
	// Host is the host where the UI should get redirected
	// +optional
	Host string `json:"host,omitempty"`

	// Signature is the signature of the agent for the host
	// +optional
	Signature string `json:"signature,omitempty"`
}

// IngressAuthTokenStatus holds the object status
type IngressAuthTokenStatus struct {
	// +optional
	Token string `json:"token,omitempty"`
}
