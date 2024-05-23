package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SharedSecret holds the secret information
// +k8s:openapi-gen=true
// +resource:path=sharedsecrets,rest=SharedSecretREST
type SharedSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SharedSecretSpec   `json:"spec,omitempty"`
	Status SharedSecretStatus `json:"status,omitempty"`
}

// SharedSecretSpec holds the specification
type SharedSecretSpec struct {
	storagev1.SharedSecretSpec `json:",inline"`
}

// SharedSecretStatus holds the status
type SharedSecretStatus struct {
	storagev1.SharedSecretStatus `json:",inline"`
}

func (a *SharedSecret) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *SharedSecret) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *SharedSecret) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *SharedSecret) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
