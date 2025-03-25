package v1

import (
	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +genclient:method=Up,verb=create,subresource=up,input=github.com/loft-sh/api/v4/pkg/apis/management/v1.DevPodWorkspaceInstanceUp,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.DevPodWorkspaceInstanceUp
// +genclient:method=Stop,verb=create,subresource=stop,input=github.com/loft-sh/api/v4/pkg/apis/management/v1.DevPodWorkspaceInstanceStop,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.DevPodWorkspaceInstanceStop
// +genclient:method=Troubleshoot,verb=get,subresource=troubleshoot,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.DevPodWorkspaceInstanceTroubleshoot
// +genclient:method=Cancel,verb=create,subresource=cancel,input=github.com/loft-sh/api/v4/pkg/apis/management/v1.DevPodWorkspaceInstanceCancel,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.DevPodWorkspaceInstanceCancel
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DevPodWorkspaceInstance holds the DevPodWorkspaceInstance information
// +k8s:openapi-gen=true
// +resource:path=devpodworkspaceinstances,rest=DevPodWorkspaceInstanceREST
// +subresource:request=DevPodWorkspaceInstanceUp,path=up,kind=DevPodWorkspaceInstanceUp,rest=DevPodWorkspaceInstanceUpREST
// +subresource:request=DevPodWorkspaceInstanceStop,path=stop,kind=DevPodWorkspaceInstanceStop,rest=DevPodWorkspaceInstanceStopREST
// +subresource:request=DevPodWorkspaceInstanceTroubleshoot,path=troubleshoot,kind=DevPodWorkspaceInstanceTroubleshoot,rest=DevPodWorkspaceInstanceTroubleshootREST
// +subresource:request=DevPodWorkspaceInstanceLog,path=log,kind=DevPodWorkspaceInstanceLog,rest=DevPodWorkspaceInstanceLogREST
// +subresource:request=DevPodWorkspaceInstanceTasks,path=tasks,kind=DevPodWorkspaceInstanceTasks,rest=DevPodWorkspaceInstanceTasksREST
// +subresource:request=DevPodWorkspaceInstanceCancel,path=cancel,kind=DevPodWorkspaceInstanceCancel,rest=DevPodWorkspaceInstanceCancelREST
// +subresource:request=DevPodWorkspaceInstanceDownload,path=download,kind=DevPodWorkspaceInstanceDownload,rest=DevPodWorkspaceInstanceDownloadREST
type DevPodWorkspaceInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevPodWorkspaceInstanceSpec   `json:"spec,omitempty"`
	Status DevPodWorkspaceInstanceStatus `json:"status,omitempty"`
}

// DevPodWorkspaceInstanceSpec holds the specification
type DevPodWorkspaceInstanceSpec struct {
	storagev1.DevPodWorkspaceInstanceSpec `json:",inline"`
}

// DevPodWorkspaceInstanceStatus holds the status
type DevPodWorkspaceInstanceStatus struct {
	storagev1.DevPodWorkspaceInstanceStatus `json:",inline"`

	// SleepModeConfig is the sleep mode config of the workspace. This will only be shown
	// in the front end.
	// +optional
	SleepModeConfig *clusterv1.SleepModeConfig `json:"sleepModeConfig,omitempty"`
}

func (a *DevPodWorkspaceInstance) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *DevPodWorkspaceInstance) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *DevPodWorkspaceInstance) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *DevPodWorkspaceInstance) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *DevPodWorkspaceInstance) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *DevPodWorkspaceInstance) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
