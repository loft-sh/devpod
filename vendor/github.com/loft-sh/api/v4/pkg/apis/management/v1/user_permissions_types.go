package v1

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type UserPermissions struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// ClusterRoles that apply to the user.
	// +optional
	ClusterRoles []UserPermissionsRole `json:"clusterRoles,omitempty"`

	// NamespaceRoles that apply to the user. Can be either regular roles or cluster roles that are namespace scoped.
	// +optional
	NamespaceRoles []UserPermissionsRole `json:"namespaceRoles,omitempty"`

	// TeamMemberships gives information about the user's team membership
	// +optional
	TeamMemberships []ObjectName `json:"teamMemberships,omitempty"`

	// ProjectMemberships gives information about the user's project membership
	ProjectMemberships []ProjectMembership `json:"projectMemberships,omitempty"`

	// ManagementRoles gives information about the user's assigned management roles
	ManagementRoles []ManagementRole `json:"managementRoles,omitempty"`

	// ClustersAccessRoles gives information about the user's assigned cluster roles and the clusters they apply to
	ClusterAccessRoles []ClusterAccessRole `json:"clusterAccessRoles,omitempty"`

	// VirtualClusterRoles give information about the user's cluster role within the virtual cluster
	VirtualClusterRoles []VirtualClusterRole `json:"virtualClusterRoles,omitempty"`
}

type UserPermissionsRole struct {
	// ClusterRole is the name of the cluster role assigned
	// to this user.
	// +optional
	ClusterRole string `json:"clusterRole,omitempty"`

	// Role is the name of the role assigned to this user.
	// +optional
	Role string `json:"role,omitempty"`

	// Namespace where this rules are valid.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Rules are the roles rules
	// +optional
	Rules []rbacv1.PolicyRule `json:"rules,omitempty"`
}

type ObjectName struct {
	// Namespace of the referenced object
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Name of the referenced object
	// +optional
	Name string `json:"name,omitempty"`

	// DisplayName is the name of the object to display in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`
}

type ProjectMembership struct {
	// ObjectName describes the project
	ObjectName `json:",inline"`

	// Role is the role given to the member
	// +optional
	Role ProjectRole `json:"role,omitempty"`

	// AssignedVia describes the resource that establishes the project membership
	// +optional
	AssignedVia AssignedVia `json:"assignedVia,omitempty"`
}

type ProjectRole struct {
	// ObjectName describes the role
	ObjectName `json:",inline"`

	// IsAdmin describes whether this is an admin project role
	// +optional
	IsAdmin bool `json:"isAdmin,omitempty"`
}

type ManagementRole struct {
	// ObjectName describes the role
	ObjectName `json:",inline"`

	// AssignedVia describes the resource that establishes the project membership
	// +optional
	AssignedVia AssignedVia `json:"assignedVia,omitempty"`
}

type ClusterAccessRole struct {
	// ObjectName describes the role
	ObjectName `json:",inline"`

	// Clusters are the clusters that this assigned role applies
	Clusters []ObjectName `json:"clusters,omitempty"`

	// AssignedVia describes the resource that establishes the project membership
	// +optional
	AssignedVia AssignedVia `json:"assignedVia,omitempty"`
}

type VirtualClusterRole struct {
	// ObjectName describes the virtual cluster
	ObjectName `json:",inline"`

	// Role is the cluster role inside the virtual cluster. One of cluster-admin, admin, edit, or view
	Role string `json:"role,omitempty"`

	// AssignedVia describes the resource that establishes the project membership
	// +optional
	AssignedVia AssignedVia `json:"assignedVia,omitempty"`
}

type AssignedVia struct {
	// ObjectName describes the name of the resource used to establish the assignment.
	ObjectName `json:",inline"`

	// Kind is the type of resource used to establish the assignment.
	// One of `User`, `Team`, or `ClusterAccess`
	// +optional
	Kind string `json:"kind,omitempty"`

	// Owner indicates if the
	Owner bool `json:"owner,omitempty"`
}
