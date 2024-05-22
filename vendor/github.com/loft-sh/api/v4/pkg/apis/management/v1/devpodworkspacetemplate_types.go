package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DevPodWorkspaceTemplate holds the information
// +k8s:openapi-gen=true
// +resource:path=devpodworkspacetemplates,rest=DevPodWorkspaceTemplateREST
type DevPodWorkspaceTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevPodWorkspaceTemplateSpec   `json:"spec,omitempty"`
	Status DevPodWorkspaceTemplateStatus `json:"status,omitempty"`
}

// DevPodWorkspaceTemplateSpec holds the specification
type DevPodWorkspaceTemplateSpec struct {
	storagev1.DevPodWorkspaceTemplateSpec `json:",inline"`
}

// DevPodWorkspaceTemplateStatus holds the status
type DevPodWorkspaceTemplateStatus struct {
	storagev1.DevPodWorkspaceTemplateStatus `json:",inline"`
}

func (a *DevPodWorkspaceTemplate) GetVersions() []storagev1.VersionAccessor {
	var retVersions []storagev1.VersionAccessor
	for _, v := range a.Spec.Versions {
		b := v
		retVersions = append(retVersions, &b)
	}

	return retVersions
}

func (a *DevPodWorkspaceTemplate) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *DevPodWorkspaceTemplate) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *DevPodWorkspaceTemplate) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *DevPodWorkspaceTemplate) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
