package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// User holds the user information
// +k8s:openapi-gen=true
type Team struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamSpec   `json:"spec,omitempty"`
	Status TeamStatus `json:"status,omitempty"`
}

func (a *Team) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *Team) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *Team) GetAccess() []Access {
	return a.Spec.Access
}

func (a *Team) SetAccess(access []Access) {
	a.Spec.Access = access
}

type TeamSpec struct {
	// The display name shown in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes a cluster access object
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// The username of the team that will be used for identification and docker registry namespace
	// +optional
	Username string `json:"username,omitempty"`

	// The loft users that belong to a team
	// +optional
	Users []string `json:"users,omitempty"`

	// The groups defined in a token that belong to a team
	// +optional
	Groups []string `json:"groups,omitempty"`

	// ImagePullSecrets holds secret references to image pull
	// secrets the team has access to.
	// +optional
	ImagePullSecrets []*KindSecretRef `json:"imagePullSecrets,omitempty"`

	// ClusterRoles define the cluster roles that the users should have assigned in the cluster.
	// +optional
	ClusterRoles []ClusterRoleRef `json:"clusterRoles,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`
}

type TeamStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TeamList contains a list of Team
type TeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Team `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Team{}, &TeamList{})
}
