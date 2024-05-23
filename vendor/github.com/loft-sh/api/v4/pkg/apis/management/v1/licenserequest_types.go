package v1

import (
	"github.com/loft-sh/admin-apis/pkg/licenseapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LicenseRequest holds license request information
// +subresource-request
type LicenseRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the admin request spec (the input for the request).
	Spec LicenseRequestSpec `json:"spec,omitempty"`

	// Status is the admin request output (the output or result of the request).
	Status LicenseRequestStatus `json:"status,omitempty"`
}

type LicenseRequestSpec struct {
	// URL is the url for the request.
	URL string `json:"url,omitempty"`

	// Input is the input payload to send to the url.
	Input licenseapi.GenericRequestInput `json:"input,omitempty"`
}

type LicenseRequestStatus struct {
	// Output is where the request output is stored.
	// +optional
	Output *licenseapi.GenericRequestOutput `json:"output,omitempty"`
}
