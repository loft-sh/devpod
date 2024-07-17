package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// User holds the user information
// +k8s:openapi-gen=true
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

func (a *User) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *User) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *User) GetAccess() []Access {
	return a.Spec.Access
}

func (a *User) SetAccess(access []Access) {
	a.Spec.Access = access
}

type UserSpec struct {
	// The display name shown in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes a cluster access object
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// The username that is used to login
	Username string `json:"username,omitempty"`

	// The URL to an icon that should be shown for the user
	// +optional
	Icon string `json:"icon,omitempty"`

	// The users email address
	// +optional
	Email string `json:"email,omitempty"`

	// The user subject as presented by the token
	Subject string `json:"subject,omitempty"`

	// The groups the user has access to
	// +optional
	Groups []string `json:"groups,omitempty"`

	// SSOGroups is used to remember groups that were
	// added from sso.
	// +optional
	SSOGroups []string `json:"ssoGroups,omitempty"`

	// A reference to the user password
	// +optional
	PasswordRef *SecretRef `json:"passwordRef,omitempty"`

	// A reference to the users access keys
	// +optional
	CodesRef *SecretRef `json:"codesRef,omitempty"`

	// ImagePullSecrets holds secret references to image pull
	// secrets the user has access to.
	// +optional
	ImagePullSecrets []*KindSecretRef `json:"imagePullSecrets,omitempty"`

	// TokenGeneration can be used to invalidate all user tokens
	// +optional
	TokenGeneration int64 `json:"tokenGeneration,omitempty"`

	// If disabled is true, an user will not be able to login anymore. All other user resources
	// are unaffected and other users can still interact with this user
	// +optional
	Disabled bool `json:"disabled,omitempty"`

	// ClusterRoles define the cluster roles that the users should have assigned in the cluster.
	// +optional
	ClusterRoles []ClusterRoleRef `json:"clusterRoles,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`
}

// UserStatus holds the status of an user
type UserStatus struct {
	// Teams the user is currently part of
	// +optional
	Teams []string `json:"teams,omitempty"`
}

// KindSecretRef is the reference to a secret containing the user password
type KindSecretRef struct {
	// APIGroup is the api group of the secret
	APIGroup string `json:"apiGroup,omitempty"`
	// Kind is the kind of the secret
	Kind string `json:"kind,omitempty"`
	// +optional
	SecretName string `json:"secretName,omitempty"`
	// +optional
	SecretNamespace string `json:"secretNamespace,omitempty"`
	// +optional
	Key string `json:"key,omitempty"`
}

// SecretRef is the reference to a secret containing the user password
type SecretRef struct {
	// +optional
	SecretName string `json:"secretName,omitempty"`
	// +optional
	SecretNamespace string `json:"secretNamespace,omitempty"`
	// +optional
	Key string `json:"key,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UserList contains a list of User
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
