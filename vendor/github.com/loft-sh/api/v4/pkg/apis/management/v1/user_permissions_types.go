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
