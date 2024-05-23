package v1

import (
	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SpaceInstance holds the SpaceInstance information
// +k8s:openapi-gen=true
// +resource:path=spaceinstances,rest=SpaceInstanceREST
type SpaceInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpaceInstanceSpec   `json:"spec,omitempty"`
	Status SpaceInstanceStatus `json:"status,omitempty"`
}

// SpaceInstanceSpec holds the specification
type SpaceInstanceSpec struct {
	storagev1.SpaceInstanceSpec `json:",inline"`
}

// SpaceInstanceStatus holds the status
type SpaceInstanceStatus struct {
	storagev1.SpaceInstanceStatus `json:",inline"`

	// SleepModeConfig is the sleep mode config of the space. This will only be shown
	// in the front end.
	// +optional
	SleepModeConfig *clusterv1.SleepModeConfig `json:"sleepModeConfig,omitempty"`

	// CanUse specifies if the requester can use the instance
	// +optional
	CanUse bool `json:"canUse,omitempty"`

	// CanUpdate specifies if the requester can update the instance
	// +optional
	CanUpdate bool `json:"canUpdate,omitempty"`
}

func (a *SpaceInstance) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *SpaceInstance) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *SpaceInstance) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *SpaceInstance) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *SpaceInstance) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *SpaceInstance) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
