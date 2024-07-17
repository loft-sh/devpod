package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type ClusterMembers struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Teams holds all the teams that have access to the cluster
	Teams []ClusterMember `json:"teams,omitempty"`

	// Users holds all the users that have access to the cluster
	Users []ClusterMember `json:"users,omitempty"`
}

type ClusterMember struct {
	// Info about the user or team
	// +optional
	Info storagev1.EntityInfo `json:"info,omitempty"`
}
