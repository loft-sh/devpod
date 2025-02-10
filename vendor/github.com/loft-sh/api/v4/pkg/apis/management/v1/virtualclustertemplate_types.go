package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterTemplate holds the information
// +k8s:openapi-gen=true
// +resource:path=virtualclustertemplates,rest=VirtualClusterTemplateREST
type VirtualClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterTemplateSpec   `json:"spec,omitempty"`
	Status VirtualClusterTemplateStatus `json:"status,omitempty"`
}

// VirtualClusterTemplateSpec holds the specification
type VirtualClusterTemplateSpec struct {
	storagev1.VirtualClusterTemplateSpec `json:",inline"`
}

// VirtualClusterTemplateStatus holds the status
type VirtualClusterTemplateStatus struct {
	storagev1.VirtualClusterTemplateStatus `json:",inline"`

	// +optional
	Apps []*storagev1.EntityInfo `json:"apps,omitempty"`
}

func (a *VirtualClusterTemplate) GetVersions() []storagev1.VersionAccessor {
	var retVersions []storagev1.VersionAccessor
	for _, v := range a.Spec.Versions {
		b := v
		retVersions = append(retVersions, &b)
	}

	return retVersions
}

func (a *VirtualClusterTemplate) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *VirtualClusterTemplate) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *VirtualClusterTemplate) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *VirtualClusterTemplate) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
