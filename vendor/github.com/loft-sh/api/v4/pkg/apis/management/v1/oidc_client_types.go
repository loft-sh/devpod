package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OIDCClient represents an OIDC client to use with Loft as an OIDC provider
// +k8s:openapi-gen=true
// +resource:path=oidcclients,rest=OIDCClientREST
type OIDCClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OIDCClientSpec   `json:"spec,omitempty"`
	Status OIDCClientStatus `json:"status,omitempty"`
}

// OIDCClientSpec holds the specification
type OIDCClientSpec struct {
	// The client name
	Name string `json:"name,omitempty"`

	// The client id of the client
	ClientID string `json:"clientId,omitempty"`

	// The client secret of the client
	ClientSecret string `json:"clientSecret,omitempty"`

	// A registered set of redirect URIs. When redirecting from dex to the client, the URI
	// requested to redirect to MUST match one of these values, unless the client is "public".
	RedirectURIs []string `json:"redirectURIs"`
}

// OIDCClientStatus holds the status
type OIDCClientStatus struct {
}
