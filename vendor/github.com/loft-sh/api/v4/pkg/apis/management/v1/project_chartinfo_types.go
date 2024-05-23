package v1

import (
	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type ProjectChartInfo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectChartInfoSpec   `json:"spec,omitempty"`
	Status ProjectChartInfoStatus `json:"status,omitempty"`
}

type ProjectChartInfoSpec struct {
	clusterv1.ChartInfoSpec `json:",inline"`
}

type ProjectChartInfoStatus struct {
	clusterv1.ChartInfoStatus `json:",inline"`
}
