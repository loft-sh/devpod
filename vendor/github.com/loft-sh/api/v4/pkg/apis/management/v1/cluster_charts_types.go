package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type ClusterCharts struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Holds the available helm charts for this cluster
	Charts []storagev1.HelmChart `json:"charts"`

	// Busy will indicate if the chart parsing is still
	// in progress.
	// +optional
	Busy bool `json:"busy,omitempty"`
}
