package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type ClusterVirtualClusterDefaults struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// DefaultTemplate is the default virtual cluster template
	// +optional
	DefaultTemplate *storagev1.VirtualClusterTemplate `json:"defaultTemplate,omitempty"`

	// LatestVersion is the latest virtual cluster version
	// +optional
	LatestVersion string `json:"latestVersion,omitempty"`

	// Default values for the virtual cluster chart
	// +optional
	Values string `json:"values,omitempty"`

	// Warning should be somehow shown to the user when
	// there is a problem retrieving the defaults
	// +optional
	Warning string `json:"warning,omitempty"`
}
