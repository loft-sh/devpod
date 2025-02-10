package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterRoleTemplate holds the clusterRoleTemplate information
// +k8s:openapi-gen=true
// +resource:path=clusterroletemplates,rest=ClusterRoleTemplateREST
type ClusterRoleTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRoleTemplateSpec   `json:"spec,omitempty"`
	Status ClusterRoleTemplateStatus `json:"status,omitempty"`
}

// ClusterRoleTemplateSpec holds the specification
type ClusterRoleTemplateSpec struct {
	storagev1.ClusterRoleTemplateSpec `json:",inline"`
}

// ClusterRoleTemplateStatus holds the status
type ClusterRoleTemplateStatus struct {
	storagev1.ClusterRoleTemplateStatus `json:",inline"`

	// +optional
	Clusters []*storagev1.EntityInfo `json:"clusters,omitempty"`
}

func (a *ClusterRoleTemplate) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *ClusterRoleTemplate) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *ClusterRoleTemplate) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *ClusterRoleTemplate) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
