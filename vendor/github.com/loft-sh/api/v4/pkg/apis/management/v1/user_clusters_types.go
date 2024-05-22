package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type UserClusters struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Clusters []ClusterAccounts `json:"clusters,omitempty"`
}

type ClusterAccounts struct {
	// Accounts are the accounts that belong to the user in the cluster
	Accounts []string `json:"accounts,omitempty"`

	// Cluster is the cluster object
	Cluster storagev1.Cluster `json:"cluster,omitempty"`
}
