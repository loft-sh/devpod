package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SharedSecret holds the secret information
// +k8s:openapi-gen=true
type SharedSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SharedSecretSpec   `json:"spec,omitempty"`
	Status SharedSecretStatus `json:"status,omitempty"`
}

// GetConditions implements conditions.Setter
func (a *SharedSecret) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

// SetConditions implements conditions.Setter
func (a *SharedSecret) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *SharedSecret) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *SharedSecret) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *SharedSecret) GetAccess() []Access {
	return a.Spec.Access
}

func (a *SharedSecret) SetAccess(access []Access) {
	a.Spec.Access = access
}

// SharedSecretSpec holds the specification
type SharedSecretSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes a shared secret
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Data contains the secret data. Each key must consist of alphanumeric
	// characters, '-', '_' or '.'. The serialized form of the secret data is a
	// base64 encoded string, representing the arbitrary (possibly non-string)
	// data value here. Described in https://tools.ietf.org/html/rfc4648#section-4
	// +optional
	Data map[string][]byte `json:"data,omitempty"`

	// Access holds the access rights for users and teams which will be transformed
	// to Roles and RoleBindings
	// +optional
	Access []Access `json:"access,omitempty"`
}

// Access describes the access to a secret
type Access struct {
	// Name is an optional name that is used for this access rule
	// +optional
	Name string `json:"name,omitempty"`

	// Verbs is a list of Verbs that apply to ALL the ResourceKinds and AttributeRestrictions contained in this rule. VerbAll represents all kinds.
	Verbs []string `json:"verbs"`

	// Subresources defines the sub resources that are allowed by this access rule
	// +optional
	Subresources []string `json:"subresources,omitempty"`

	// Users specifies which users should be able to access this secret with the aforementioned verbs
	// +optional
	Users []string `json:"users,omitempty"`

	// Teams specifies which teams should be able to access this secret with the aforementioned verbs
	// +optional
	Teams []string `json:"teams,omitempty"`
}

// SharedSecretStatus holds the status
type SharedSecretStatus struct {
	// Conditions holds several conditions the project might be in
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SharedSecretList contains a list of SharedSecret
type SharedSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SharedSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SharedSecret{}, &SharedSecretList{})
}
