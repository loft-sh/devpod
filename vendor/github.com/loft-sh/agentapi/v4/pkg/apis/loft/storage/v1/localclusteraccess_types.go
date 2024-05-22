package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LocalClusterAccess holds the cluster access information
// +k8s:openapi-gen=true
type LocalClusterAccess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LocalClusterAccessSpec   `json:"spec,omitempty"`
	Status LocalClusterAccessStatus `json:"status,omitempty"`
}

type LocalClusterAccessSpec struct {
	// DisplayName is the name that should be shown in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description is the description of this object in
	// human-readable text.
	// +optional
	Description string `json:"description,omitempty"`

	// Users are the users affected by this cluster access object
	// +optional
	Users []UserOrTeam `json:"users,omitempty"`

	// Teams are the teams affected by this cluster access object
	// +optional
	Teams []string `json:"teams,omitempty"`

	// ClusterRoles define the cluster roles that the users should have assigned in the cluster.
	// +optional
	ClusterRoles []ClusterRoleRef `json:"clusterRoles,omitempty"`

	// Priority is a unique value that specifies the priority of this cluster access
	// for the space constraints and quota. A higher priority means the cluster access object
	// will override the space constraints of lower priority cluster access objects
	// +optional
	Priority int `json:"priority,omitempty"`

	// SpaceConstraintsRef is a reference to a space constraints object
	// +optional
	SpaceConstraintsRef *string `json:"spaceConstraintsRef,omitempty"`

	// Quota defines the quotas for the members that should be created.
	// +optional
	Quota *AccessQuota `json:"quota,omitempty"`
}

type AccessQuota struct {
	// hard is the set of desired hard limits for each named resource.
	// More info: https://kubernetes.io/docs/concepts/policy/resource-quotas/
	// +optional
	Hard corev1.ResourceList `json:"hard,omitempty"`
}

type ClusterRoleRef struct {
	// Name is the cluster role to assign
	// +optional
	Name string `json:"name,omitempty"`
}

type UserOrTeam struct {
	// Name of a Loft user
	// +optional
	User string `json:"user,omitempty"`

	// Name of a Loft team the user is part of
	// +optional
	Team string `json:"team,omitempty"`
}

// LocalClusterAccessStatus holds the status of a user access
type LocalClusterAccessStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LocalClusterAccessList contains a list of cluster access objects
type LocalClusterAccessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LocalClusterAccess `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LocalClusterAccess{}, &LocalClusterAccessList{})
}
