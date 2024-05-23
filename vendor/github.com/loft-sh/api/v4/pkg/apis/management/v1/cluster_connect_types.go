package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterConnect holds the information
// +k8s:openapi-gen=true
// +resource:path=clusterconnect,rest=ClusterConnectREST
type ClusterConnect struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterConnectSpec   `json:"spec,omitempty"`
	Status ClusterConnectStatus `json:"status,omitempty"`
}

type ClusterConnectSpec struct {
	// the kube config used to connect the cluster
	// +optional
	Config string `json:"config,omitempty"`

	// The user to create an admin account for
	// +optional
	AdminUser string `json:"adminUser,omitempty"`

	// the cluster template to create
	ClusterTemplate Cluster `json:"clusterTemplate,omitempty"`
}

type ClusterConnectStatus struct {
	// +optional
	Failed bool `json:"failed,omitempty"`

	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`
}
