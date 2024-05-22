package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConvertVirtualClusterConfig holds config request and response data for virtual clusters
// +k8s:openapi-gen=true
// +resource:path=convertvirtualclusterconfig,rest=ConvertVirtualClusterConfigREST
type ConvertVirtualClusterConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConvertVirtualClusterConfigSpec   `json:"spec,omitempty"`
	Status ConvertVirtualClusterConfigStatus `json:"status,omitempty"`
}

// ConvertVirtualClusterConfigSpec holds the specification
type ConvertVirtualClusterConfigSpec struct {
	// Distro is the distro to be used for the config
	// +optional
	Distro string `json:"distro,omitempty"`

	// Values are the config values for the virtual cluster
	// +optional
	Values string `json:"values,omitempty"`
}

// ConvertVirtualClusterConfigStatus holds the status
type ConvertVirtualClusterConfigStatus struct {
	// Values are the converted config values for the virtual cluster
	// +optional
	Values string `json:"values,omitempty"`

	// Converted signals if the Values have been converted from the old format
	Converted bool `json:"converted"`
}
