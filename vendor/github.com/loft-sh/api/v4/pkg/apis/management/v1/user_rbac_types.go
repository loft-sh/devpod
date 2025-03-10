package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type UserDetailedPermissions struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	TeamMemberships []UserDrilldownPermissionsTeam `json:"teamMemberships,omitempty"`

	// +optional
	RolesAssigned []UserDrilldownManagementRoles `json:"rolesAssigned,omitempty"`

	// +optional
	ProjectMemberships []UserDrilldownProjectMemberships `json:"projectMemberships,omitempty"`

	// +optional
	VirtualClusterRoles []UserDrilldownVClusterRoles `json:"virtualClusterRoles,omitempty"`
}

type UserDrilldownPermissionsTeam struct {
	ObjectNames `json:",omitempty"`
}

type UserDrilldownManagementRoles struct {
	ObjectNames `json:",omitempty"`
	Management  bool        `json:"management,omitempty"`
	AssignedVia AssignedVia `json:"assignedVia,omitempty"`
}

type UserDrilldownProjectMemberships struct {
	ObjectNames `json:",omitempty"`
	Role        string      `json:"role,omitempty"`
	AssignedVia AssignedVia `json:"assignedVia,omitempty"`
}

type AssignedVia struct {
	Team string `json:"team,omitempty"`
}

type UserDrilldownVClusterRoles struct {
	ObjectNames `json:",omitempty"`
	Role        string      `json:"role,omitempty"`
	AssignedVia AssignedVia `json:"assignedVia,omitempty"`
}

type ObjectNames struct {
	// +optional
	Name string `json:"name"`
	// +optional
	DisplayName string `json:"displayName,omitempty"`
}
