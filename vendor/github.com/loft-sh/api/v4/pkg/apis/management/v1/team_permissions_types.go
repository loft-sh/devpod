package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type TeamPermissions struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Members gives users that are team members
	// +optional
	Members []ObjectName `json:"members,omitempty"`

	// ProjectMemberships gives information about the team's project membership
	ProjectMemberships []ProjectMembership `json:"projectMemberships,omitempty"`

	// ManagementRoles gives information about the team's assigned management roles
	ManagementRoles []ManagementRole `json:"managementRoles,omitempty"`

	// ClustersAccessRoles gives information about the team's assigned cluster roles and the clusters they apply to
	ClusterAccessRoles []ClusterAccessRole `json:"clusterAccessRoles,omitempty"`

	// VirtualClusterRoles give information about the team's cluster role within the virtual cluster
	VirtualClusterRoles []VirtualClusterRole `json:"virtualClusterRoles,omitempty"`
}
