package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LocalTeam holds the cluster user information
// +k8s:openapi-gen=true
type LocalTeam struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LocalTeamSpec   `json:"spec,omitempty"`
	Status LocalTeamStatus `json:"status,omitempty"`
}

type LocalTeamSpec struct {
}

// LocalTeamStatus holds the status of a user access
type LocalTeamStatus struct {
	// The display name shown in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// The username of the team that will be used for identification and docker registry namespace
	// +optional
	Username string `json:"username,omitempty"`

	// The loft users that belong to a team
	// +optional
	Users []string `json:"users,omitempty"`

	// The groups defined in a token that belong to a team
	// +optional
	Groups []string `json:"groups,omitempty"`

	// Labels are the labels of the user
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are the annotations of the user
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LocalTeamList contains a list of LocalTeam objects
type LocalTeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LocalTeam `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LocalTeam{}, &LocalTeamList{})
}
