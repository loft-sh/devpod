package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TranslateVClusterResourceName holds rename request and response data for given resource
// +k8s:openapi-gen=true
// +resource:path=translatevclusterresourcenames,rest=TranslateVClusterResourceNameREST
type TranslateVClusterResourceName struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TranslateVClusterResourceNameSpec   `json:"spec,omitempty"`
	Status TranslateVClusterResourceNameStatus `json:"status,omitempty"`
}

// TranslateVClusterResourceNameSpec holds the specification
type TranslateVClusterResourceNameSpec struct {
	// Name is the name of resource we want to rename
	Name string `json:"name"`

	// Namespace is the name of namespace in which this resource is running
	Namespace string `json:"namespace"`

	// VClusterName is the name of vCluster in which this resource is running
	VClusterName string `json:"vclusterName"`
}

// TranslateVClusterResourceNameStatus holds the status
type TranslateVClusterResourceNameStatus struct {
	// Name is the converted name of resource
	// +optional
	Name string `json:"name,omitempty"`
}
