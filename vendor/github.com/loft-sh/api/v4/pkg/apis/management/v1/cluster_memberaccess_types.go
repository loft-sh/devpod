package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type ClusterMemberAccess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Teams holds all the teams that the current user has access to the cluster
	Teams []ClusterMember `json:"teams,omitempty"`

	// Users holds all the users that the current user has access to the cluster
	Users []ClusterMember `json:"users,omitempty"`
}
