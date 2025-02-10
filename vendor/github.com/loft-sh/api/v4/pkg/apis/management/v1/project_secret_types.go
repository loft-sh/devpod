package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	LoftProjectSecret                           = "loft.sh/project-secret"
	LoftProjectSecretNameLabel                  = "loft.sh/project-secret-name"
	LoftProjectSecretDescription                = "loft.sh/project-secret-description"
	LoftProjectSecretDisplayName                = "loft.sh/project-secret-displayname"
	LoftProjectSecretOwner                      = "loft.sh/project-secret-owner"
	LoftProjectSecretAccess                     = "loft.sh/project-secret-access"
	LoftProjectSecretStatusConditionsAnnotation = "loft.sh/project-secret-status-conditions"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectSecret holds the Project Secret information
// +k8s:openapi-gen=true
// +resource:path=projectsecrets,rest=ProjectSecretREST
type ProjectSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSecretSpec   `json:"spec,omitempty"`
	Status ProjectSecretStatus `json:"status,omitempty"`
}

func (a *ProjectSecret) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *ProjectSecret) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *ProjectSecret) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *ProjectSecret) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}

// ProjectSecretSpec holds the specification
type ProjectSecretSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes a Project secret
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *storagev1.UserOrTeam `json:"owner,omitempty"`

	// Data contains the secret data. Each key must consist of alphanumeric
	// characters, '-', '_' or '.'. The serialized form of the secret data is a
	// base64 encoded string, representing the arbitrary (possibly non-string)
	// data value here. Described in https://tools.ietf.org/html/rfc4648#section-4
	// +optional
	Data map[string][]byte `json:"data,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []storagev1.Access `json:"access,omitempty"`
}

// ProjectSecretStatus holds the status
type ProjectSecretStatus struct {
	// Conditions holds several conditions the project might be in
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`
}
