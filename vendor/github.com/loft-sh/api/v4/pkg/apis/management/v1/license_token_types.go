package v1

import (
	"github.com/loft-sh/admin-apis/pkg/licenseapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// License Token holds the license token information
// +k8s:openapi-gen=true
// +resource:path=licensetokens,rest=LicenseTokenREST
type LicenseToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LicenseTokenSpec   `json:"spec,omitempty"`
	Status LicenseTokenStatus `json:"status,omitempty"`
}

type LicenseTokenSpec struct {
	URL     string `json:"url,omitempty"`
	Payload string `json:"payload,omitempty"`
}

type LicenseTokenStatus struct {
	Token *licenseapi.InstanceTokenAuth `json:"token,omitempty"`
}
