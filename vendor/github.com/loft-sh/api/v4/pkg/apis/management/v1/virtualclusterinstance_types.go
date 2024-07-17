package v1

import (
	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +genclient:method=GetKubeConfig,verb=create,subresource=kubeconfig,input=github.com/loft-sh/api/v4/pkg/apis/management/v1.VirtualClusterInstanceKubeConfig,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.VirtualClusterInstanceKubeConfig
// +genclient:method=GetAccessKey,verb=get,subresource=accesskey,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.VirtualClusterAccessKey
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterInstance holds the VirtualClusterInstance information
// +k8s:openapi-gen=true
// +resource:path=virtualclusterinstances,rest=VirtualClusterInstanceREST
// +subresource:request=VirtualClusterInstanceLog,path=log,kind=VirtualClusterInstanceLog,rest=VirtualClusterInstanceLogREST
// +subresource:request=VirtualClusterInstanceKubeConfig,path=kubeconfig,kind=VirtualClusterInstanceKubeConfig,rest=VirtualClusterInstanceKubeConfigREST
// +subresource:request=VirtualClusterAccessKey,path=accesskey,kind=VirtualClusterAccessKey,rest=VirtualClusterAccessKeyREST
type VirtualClusterInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterInstanceSpec   `json:"spec,omitempty"`
	Status VirtualClusterInstanceStatus `json:"status,omitempty"`
}

// VirtualClusterInstanceSpec holds the specification
type VirtualClusterInstanceSpec struct {
	storagev1.VirtualClusterInstanceSpec `json:",inline"`
}

// VirtualClusterInstanceStatus holds the status
type VirtualClusterInstanceStatus struct {
	storagev1.VirtualClusterInstanceStatus `json:",inline"`

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

	// Online specifies if there is at least one network peer available
	// for an agentless vCluster.
	// +optional
	Online bool `json:"online,omitempty"`
}

func (a *VirtualClusterInstance) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *VirtualClusterInstance) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *VirtualClusterInstance) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *VirtualClusterInstance) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *VirtualClusterInstance) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *VirtualClusterInstance) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
