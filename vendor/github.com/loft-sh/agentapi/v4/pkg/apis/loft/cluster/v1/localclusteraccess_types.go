package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LocalClusterAccess holds the cluster access information
// +k8s:openapi-gen=true
// +resource:path=localclusteraccesses,rest=LocalClusterAccessREST
type LocalClusterAccess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LocalClusterAccessSpec   `json:"spec,omitempty"`
	Status LocalClusterAccessStatus `json:"status,omitempty"`
}

type LocalClusterAccessSpec struct {
	agentstoragev1.LocalClusterAccessSpec `json:",inline"`
}

type LocalClusterAccessStatus struct {
	agentstoragev1.LocalClusterAccessStatus `json:",inline"`

	// +optional
	Users []*UserOrTeam `json:"users,omitempty"`

	// +optional
	Teams []*EntityInfo `json:"teams,omitempty"`
}

type UserOrTeam struct {
	// User describes an user
	// +optional
	User *EntityInfo `json:"user,omitempty"`

	// Team describes a team
	// +optional
	Team *EntityInfo `json:"team,omitempty"`
}

type EntityInfo struct {
	// Name is the kubernetes name of the object
	Name string `json:"name,omitempty"`

	// The display name shown in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Icon is the icon of the user / team
	// +optional
	Icon string `json:"icon,omitempty"`

	// The username that is used to login
	// +optional
	Username string `json:"username,omitempty"`

	// The users email address
	// +optional
	Email string `json:"email,omitempty"`

	// The user subject
	// +optional
	Subject string `json:"subject,omitempty"`
}
