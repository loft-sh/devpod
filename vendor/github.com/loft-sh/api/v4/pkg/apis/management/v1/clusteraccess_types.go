package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterAccess holds the globalClusterAccess information
// +k8s:openapi-gen=true
// +resource:path=clusteraccesses,rest=ClusterAccessREST
type ClusterAccess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterAccessSpec   `json:"spec,omitempty"`
	Status ClusterAccessStatus `json:"status,omitempty"`
}

// ClusterAccessSpec holds the specification
type ClusterAccessSpec struct {
	storagev1.ClusterAccessSpec `json:",inline"`
}

// ClusterAccessStatus holds the status
type ClusterAccessStatus struct {
	storagev1.ClusterAccessStatus `json:",inline"`

	// +optional
	Clusters []*storagev1.EntityInfo `json:"clusters,omitempty"`

	// +optional
	Users []*storagev1.UserOrTeamEntity `json:"users,omitempty"`

	// +optional
	Teams []*storagev1.EntityInfo `json:"teams,omitempty"`
}

func (a *ClusterAccess) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *ClusterAccess) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *ClusterAccess) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *ClusterAccess) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
