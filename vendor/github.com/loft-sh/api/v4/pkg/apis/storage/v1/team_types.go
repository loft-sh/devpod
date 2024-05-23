package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
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

	// ClusterAccountTemplates that should be applied for the user
	// +optional
	ClusterAccountTemplates []UserClusterAccountTemplate `json:"clusterAccountTemplates,omitempty"`

	// ClusterRoles define the cluster roles that the users should have assigned in the cluster.
	// +optional
	ClusterRoles []agentstoragev1.ClusterRoleRef `json:"clusterRoles,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`
}

type TeamStatus struct {
	// Clusters holds information about which clusters the user has accounts in
	// +optional
	Clusters []AccountClusterStatus `json:"clusters,omitempty"`

	// ClusterAccountTemplates holds information about which cluster account templates were applied
	// DEPRECATED: Use status.clusters instead
	// +optional
	ClusterAccountTemplates []UserClusterAccountTemplateStatus `json:"clusterAccountTemplates,omitempty"`
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
