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

	// Deprecated: Use the Access Key CRD instead
	// A reference to the users access keys
	// +optional
	AccessKeysRef *SecretRef `json:"accessKeysRef,omitempty"`

	// A reference to the users access keys
	// +optional
	CodesRef *SecretRef `json:"codesRef,omitempty"`

	// ImagePullSecrets holds secret references to image pull
	// secrets the user has access to.
	// +optional
	ImagePullSecrets []*KindSecretRef `json:"imagePullSecrets,omitempty"`

	// ClusterAccountTemplates that should be applied for the user
	// +optional
	ClusterAccountTemplates []UserClusterAccountTemplate `json:"clusterAccountTemplates,omitempty"`

	// TokenGeneration can be used to invalidate all user tokens
	// +optional
	TokenGeneration int64 `json:"tokenGeneration,omitempty"`

	// If disabled is true, an user will not be able to login anymore. All other user resources
	// are unaffected and other users can still interact with this user
	// +optional
	Disabled bool `json:"disabled,omitempty"`

	// ClusterRoles define the cluster roles that the users should have assigned in the cluster.
	// +optional
	ClusterRoles []agentstoragev1.ClusterRoleRef `json:"clusterRoles,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`
}

type UserClusterAccountTemplate struct {
	// Name of the cluster account template to apply
	// +optional
	Name string `json:"name,omitempty"`

	// Sync defines if Loft should sync changes to the cluster account template
	// to the cluster accounts and create new accounts if new clusters match the templates.
	// +optional
	Sync bool `json:"sync,omitempty"`

	// AccountName is the name of the account that should
	// be created. Defaults to the user or team kubernetes name.
	// +optional
	AccountName string `json:"accountName,omitempty"`
}

// UserStatus holds the status of an user
type UserStatus struct {
	// Clusters holds information about which clusters the user has accounts in
	// +optional
	Clusters []AccountClusterStatus `json:"clusters,omitempty"`

	// ClusterAccountTemplates holds information about which cluster account templates were applied
	// DEPRECATED: Use status.clusters instead
	// +optional
	ClusterAccountTemplates []UserClusterAccountTemplateStatus `json:"clusterAccountTemplates,omitempty"`

	// Teams the user is currently part of
	// +optional
	Teams []string `json:"teams,omitempty"`
}

// AccountClusterStatus holds the status of an account in a cluster
type AccountClusterStatus struct {
	// Status holds the status of the account in the target cluster
	// +optional
	Status AccountClusterStatusPhase `json:"phase,omitempty"`

	// Reason describes why loft couldn't sync the account with a machine readable identifier
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes why loft couldn't sync the account in human language
	// +optional
	Message string `json:"message,omitempty"`

	// Cluster is the cluster name of the user in the cluster
	// +optional
	Cluster string `json:"cluster,omitempty"`

	// AccountsClusterTemplate status is the status of the account cluster template that was used
	// to create the cluster account
	// +optional
	AccountsClusterTemplate []AccountClusterTemplateStatus `json:"accountsClusterTemplateStatus,omitempty"`

	// Accounts is the account name of the user in the cluster
	// +optional
	Accounts []string `json:"accounts,omitempty"`
}

type AccountClusterTemplateStatus struct {
	// Name is the name of the cluster account template
	// +optional
	Name string `json:"name,omitempty"`

	// Account is the name of the account in the cluster
	// +optional
	Account string `json:"account,omitempty"`

	// AccountTemplateHash is the hash of the account template that was applied
	// +optional
	AccountTemplateHash string `json:"accountTemplateHash,omitempty"`

	// OwnsHash is the hash of the owns part of the cluster account template that was
	// applied
	// +optional
	OwnsHash string `json:"ownsHash,omitempty"`

	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`

	// Status holds the status of the account in the target cluster
	// +optional
	Status ClusterAccountTemplateStatusPhase `json:"phase,omitempty"`

	// Reason describes why loft couldn't sync the account with a machine readable identifier
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes why loft couldn't sync the account in human language
	// +optional
	Message string `json:"message,omitempty"`
}

// ClusterAccountTemplateStatusPhase describes the phase of a cluster
type ClusterAccountTemplateStatusPhase string

// These are the valid admin account types
const (
	ClusterAccountTemplateStatusPhaseSynced        ClusterAccountTemplateStatusPhase = "Synced"
	ClusterAccountTemplateStatusPhaseFailed        ClusterAccountTemplateStatusPhase = "Failed"
	ClusterAccountTemplateStatusPhaseFailedAccount ClusterAccountTemplateStatusPhase = "FailedAccount"
	ClusterAccountTemplateStatusPhaseFailedObjects ClusterAccountTemplateStatusPhase = "FailedObjects"
)

// AccountClusterStatusPhase describes the phase of a cluster
type AccountClusterStatusPhase string

// These are the valid admin account types
const (
	AccountClusterStatusPhaseSynced AccountClusterStatusPhase = "Synced"
	AccountClusterStatusPhaseFailed AccountClusterStatusPhase = "Failed"
)

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

type UserClusterAccountTemplateStatus struct {
	// Name of the cluster account template that was applied
	// +optional
	Name string `json:"name,omitempty"`

	// Clusters holds the cluster on which this template was applied
	// +optional
	Clusters []ClusterAccountTemplateClusterStatus `json:"clusters,omitempty"`

	// Status holds the status of the account in the target cluster
	// +optional
	Status ClusterAccountTemplateStatusPhase `json:"phase,omitempty"`

	// Reason describes why loft couldn't sync the account with a machine readable identifier
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes why loft couldn't sync the account in human language
	// +optional
	Message string `json:"message,omitempty"`
}

type ClusterAccountTemplateClusterStatus struct {
	// Name of the cluster where the cluster account template was applied
	// +optional
	Name string `json:"name,omitempty"`

	// Status holds the status of the account in the target cluster
	// +optional
	Status ClusterAccountTemplateClusterStatusPhase `json:"phase,omitempty"`

	// Reason describes why loft couldn't sync the account with a machine readable identifier
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes why loft couldn't sync the account in human language
	// +optional
	Message string `json:"message,omitempty"`
}

// ClusterAccountTemplateClusterStatusPhase describes the phase of a cluster account template
type ClusterAccountTemplateClusterStatusPhase string

// These are the valid account template cluster status
const (
	ClusterAccountTemplateClusterStatusPhaseCreated ClusterAccountTemplateClusterStatusPhase = "Created"
	ClusterAccountTemplateClusterStatusPhaseSkipped ClusterAccountTemplateClusterStatusPhase = "Skipped"
	ClusterAccountTemplateClusterStatusPhaseFailed  ClusterAccountTemplateClusterStatusPhase = "Failed"
)

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
