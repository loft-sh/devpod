package v1

import (
	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SpaceConstraint holds the globalSpaceConstraint information
// +k8s:openapi-gen=true
// +resource:path=spaceconstraints,rest=SpaceConstraintREST
type SpaceConstraint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpaceConstraintSpec   `json:"spec,omitempty"`
	Status SpaceConstraintStatus `json:"status,omitempty"`
}

// SpaceConstraintSpec holds the specification
type SpaceConstraintSpec struct {
	storagev1.SpaceConstraintSpec `json:",inline"`
}

// SpaceConstraintStatus holds the status
type SpaceConstraintStatus struct {
	storagev1.SpaceConstraintStatus `json:",inline"`

	// +optional
	ClusterRole *clusterv1.EntityInfo `json:"clusterRole,omitempty"`

	// +optional
	Clusters []*clusterv1.EntityInfo `json:"clusters,omitempty"`
}

func (a *SpaceConstraint) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *SpaceConstraint) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *SpaceConstraint) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *SpaceConstraint) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
