package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// +resource:path=chartinfos,rest=ChartInfoREST
type ChartInfo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChartInfoSpec   `json:"spec,omitempty"`
	Status ChartInfoStatus `json:"status,omitempty"`
}

type ChartInfoSpec struct {
	// Chart holds information about a chart that should get deployed
	// +optional
	Chart Chart `json:"chart,omitempty"`
}

type ChartInfoStatus struct {
	// Metadata provides information about a chart
	// +optional
	Metadata *Metadata `json:"metadata,omitempty"`

	// Readme is the readme of the chart
	// +optional
	Readme string `json:"readme,omitempty"`

	// Values are the default values of the chart
	// +optional
	Values string `json:"values,omitempty"`
}
