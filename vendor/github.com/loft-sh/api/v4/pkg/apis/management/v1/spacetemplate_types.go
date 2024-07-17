package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SpaceTemplate holds the information
// +k8s:openapi-gen=true
// +resource:path=spacetemplates,rest=SpaceTemplateREST
type SpaceTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpaceTemplateSpec   `json:"spec,omitempty"`
	Status SpaceTemplateStatus `json:"status,omitempty"`
}

// SpaceTemplateSpec holds the specification
type SpaceTemplateSpec struct {
	storagev1.SpaceTemplateSpec `json:",inline"`
}

// SpaceTemplateStatus holds the status
type SpaceTemplateStatus struct {
	storagev1.SpaceTemplateStatus `json:",inline"`

	// +optional
	Apps []*storagev1.EntityInfo `json:"apps,omitempty"`
}

func (a *SpaceTemplate) GetVersions() []storagev1.VersionAccessor {
	var retVersions []storagev1.VersionAccessor
	for _, v := range a.Spec.Versions {
		b := v
		retVersions = append(retVersions, &b)
	}

	return retVersions
}

func (a *SpaceTemplate) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *SpaceTemplate) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *SpaceTemplate) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *SpaceTemplate) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
