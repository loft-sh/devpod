package v1

import (
	"github.com/loft-sh/admin-apis/pkg/licenseapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Feature holds the feature information
// +k8s:openapi-gen=true
// +resource:path=features,rest=FeatureREST,statusRest=FeatureStatusREST
type Feature struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FeatureSpec   `json:"spec,omitempty"`
	Status FeatureStatus `json:"status,omitempty"`
}

// FeatureSpec holds the specification
type FeatureSpec struct {
}

// FeatureStatus holds the status
type FeatureStatus struct {
	// Feature contains all feature details (as typically returned by license service)
	licenseapi.Feature `json:",inline"`

	// Internal marks internal features that should not be shown on the license view
	// +optional
	Internal bool `json:"internal,omitempty"`

	// Used marks features that are currently used in the product
	// +optional
	Used bool `json:"used,omitempty"`
}
