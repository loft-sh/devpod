package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegisterVirtualCluster holds config request and response data for virtual clusters
// +k8s:openapi-gen=true
// +resource:path=registervirtualclusters,rest=RegisterVirtualClusterREST
type RegisterVirtualCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegisterVirtualClusterSpec   `json:"spec,omitempty"`
	Status RegisterVirtualClusterStatus `json:"status,omitempty"`
}

// RegisterVirtualClusterSpec holds the specification
type RegisterVirtualClusterSpec struct {
	// ServiceUID uniquely identifies the virtual cluster based on the service uid.
	// +optional
	ServiceUID string `json:"serviceUID,omitempty"`

	// Project is the project name the virtual cluster should be in.
	// +optional
	Project string `json:"project,omitempty"`

	// Name is the virtual cluster instance name. If the name is already taken, the platform will construct a
	// name for the vcluster based on the service uid and this name.
	// +optional
	Name string `json:"name,omitempty"`

	// ForceName specifies if the name should be used or creation will fail.
	// +optional
	ForceName bool `json:"forceName,omitempty"`

	// Chart specifies the vCluster chart.
	// +optional
	Chart string `json:"chart,omitempty"`

	// Version specifies the vCluster version.
	// +optional
	Version string `json:"version,omitempty"`

	// Values specifies the vCluster config.
	// +optional
	Values string `json:"values,omitempty"`
}

// RegisterVirtualClusterStatus holds the status
type RegisterVirtualClusterStatus struct {
	// Name is the actual name of the virtual cluster instance.
	// +optional
	Name string `json:"name,omitempty"`
}
