package v1

import (
	"github.com/loft-sh/admin-apis/pkg/licenseapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:method=LicenseRequest,verb=create,subresource=request,input=github.com/loft-sh/api/v4/pkg/apis/management/v1.LicenseRequest,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.LicenseRequest
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// License holds the license information
// +k8s:openapi-gen=true
// +resource:path=licenses,rest=LicenseREST
// +subresource:request=LicenseRequest,path=request,kind=LicenseRequest,rest=LicenseRequestREST
type License struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LicenseSpec   `json:"spec,omitempty"`
	Status LicenseStatus `json:"status,omitempty"`
}

type LicenseSpec struct {
}

type LicenseStatus struct {
	// License is the license data received from the license server.
	// +optional
	License *licenseapi.License `json:"license,omitempty"`

	// ResourceUsage shows the current usage of license limit.
	// +optional
	ResourceUsage map[string]licenseapi.ResourceCount `json:"resourceUsage,omitempty"`
}
