package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DevPodEnvironmentTemplate holds the DevPodEnvironmentTemplate information
// +k8s:openapi-gen=true
// +resource:path=devpodenvironmenttemplates,rest=DevPodEnvironmentTemplateREST
type DevPodEnvironmentTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevPodEnvironmentTemplateSpec   `json:"spec,omitempty"`
	Status DevPodEnvironmentTemplateStatus `json:"status,omitempty"`
}

// DevPodEnvironmentTemplateSpec holds the specification
type DevPodEnvironmentTemplateSpec struct {
	storagev1.DevPodEnvironmentTemplateSpec `json:",inline"`
}

// GitEnvironmentTemplate stores configuration of Git environment template source
type GitEnvironmentTemplate struct {
	// Repository stores repository URL for Git environment spec source
	Repository string `json:"repository"`

	// Revision stores revision to checkout in repository
	// +optional
	Revision string `json:"revision,omitempty"`

	// SubPath stores subpath within Repositor where environment spec is
	// +optional
	SubPath string `json:"subpath,omitempty"`
}

func (a *DevPodEnvironmentTemplate) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *DevPodEnvironmentTemplate) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *DevPodEnvironmentTemplate) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *DevPodEnvironmentTemplate) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}

// DevPodEnvironmentTemplateStatus holds the status
type DevPodEnvironmentTemplateStatus struct{}
