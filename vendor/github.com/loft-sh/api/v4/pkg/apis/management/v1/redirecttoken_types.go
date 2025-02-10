package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RedirectToken holds the object information
// +k8s:openapi-gen=true
// +resource:path=redirecttokens,rest=RedirectTokenREST
type RedirectToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedirectTokenSpec   `json:"spec,omitempty"`
	Status RedirectTokenStatus `json:"status,omitempty"`
}

// RedirectTokenSpec holds the object specification
type RedirectTokenSpec struct {
	// Token is the token that includes the redirect request
	// +optional
	Token string `json:"token,omitempty"`
}

// RedirectTokenStatus holds the object status
type RedirectTokenStatus struct {
	// +optional
	RedirectURL string `json:"redirectURL,omitempty"`
}

// RedirectTokenClaims holds the private claims of the redirect token
type RedirectTokenClaims struct {
	// URL is the url to redirect to.
	// +optional
	URL string `json:"url,omitempty"`
}
