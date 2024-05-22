package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LocalUser holds the cluster user information
// +k8s:openapi-gen=true
type LocalUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LocalUserSpec   `json:"spec,omitempty"`
	Status LocalUserStatus `json:"status,omitempty"`
}

type LocalUserSpec struct{}

// LocalUserStatus holds the status of a user access
type LocalUserStatus struct {
	// The display name shown in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// The username that is used to login
	Username string `json:"username,omitempty"`

	// The users email address
	// +optional
	Email string `json:"email,omitempty"`

	// The user subject as presented by the token
	Subject string `json:"subject,omitempty"`

	// The groups the user has access to
	// +optional
	Groups []string `json:"groups,omitempty"`

	// Teams the user is currently part of
	// +optional
	Teams []string `json:"teams,omitempty"`

	// Labels are the labels of the user
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are the annotations of the user
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LocalUserList contains a list of LocalUser objects
type LocalUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LocalUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LocalUser{}, &LocalUserList{})
}
