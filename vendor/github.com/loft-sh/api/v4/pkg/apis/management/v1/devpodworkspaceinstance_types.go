package v1

import (
	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
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

	// Kubernetes is the status of the workspace on kubernetes
	// +optional
	Kubernetes *DevPodWorkspaceInstanceKubernetesStatus `json:"kubernetes,omitempty"`
}

type DevPodWorkspaceInstanceKubernetesStatus struct {
	// PodStatus is the status of the pod that is running the workspace
	// +optional
	PodStatus *DevPodWorkspaceInstancePodStatus `json:"podStatus,omitempty"`

	// PersistentVolumeClaimStatus is the pvc that is used to store the workspace
	// +optional
	PersistentVolumeClaimStatus *DevPodWorkspaceInstancePersistentVolumeClaimStatus `json:"persistentVolumeClaimStatus,omitempty"`
}

type DevPodWorkspaceInstancePodStatus struct {
	// The phase of a Pod is a simple, high-level summary of where the Pod is in its lifecycle.
	// The conditions array, the reason and message fields, and the individual container status
	// arrays contain more detail about the pod's status.
	// There are five possible phase values:
	//
	// Pending: The pod has been accepted by the Kubernetes system, but one or more of the
	// container images has not been created. This includes time before being scheduled as
	// well as time spent downloading images over the network, which could take a while.
	// Running: The pod has been bound to a node, and all of the containers have been created.
	// At least one container is still running, or is in the process of starting or restarting.
	// Succeeded: All containers in the pod have terminated in success, and will not be restarted.
	// Failed: All containers in the pod have terminated, and at least one container has
	// terminated in failure. The container either exited with non-zero status or was terminated
	// by the system.
	// Unknown: For some reason the state of the pod could not be obtained, typically due to an
	// error in communicating with the host of the pod.
	//
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#pod-phase
	// +optional
	Phase corev1.PodPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=PodPhase"`
	// Current service state of pod.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#pod-conditions
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []corev1.PodCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,2,rep,name=conditions"`
	// A human readable message indicating details about why the pod is in this condition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`
	// A brief CamelCase message indicating details about why the pod is in this state.
	// e.g. 'Evicted'
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,4,opt,name=reason"`
	// Statuses of init containers in this pod. The most recent successful non-restartable
	// init container will have ready = true, the most recently started container will have
	// startTime set.
	// Each init container in the pod should have at most one status in this list,
	// and all statuses should be for containers in the pod.
	// However this is not enforced.
	// If a status for a non-existent container is present in the list, or the list has duplicate names,
	// the behavior of various Kubernetes components is not defined and those statuses might be
	// ignored.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-and-container-status
	// +listType=atomic
	InitContainerStatuses []corev1.ContainerStatus `json:"initContainerStatuses,omitempty" protobuf:"bytes,10,rep,name=initContainerStatuses"`
	// Statuses of containers in this pod.
	// Each container in the pod should have at most one status in this list,
	// and all statuses should be for containers in the pod.
	// However this is not enforced.
	// If a status for a non-existent container is present in the list, or the list has duplicate names,
	// the behavior of various Kubernetes components is not defined and those statuses might be
	// ignored.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#pod-and-container-status
	// +optional
	// +listType=atomic
	ContainerStatuses []corev1.ContainerStatus `json:"containerStatuses,omitempty" protobuf:"bytes,8,rep,name=containerStatuses"`
	// NodeName is the name of the node that is running the workspace
	// +optional
	NodeName string `json:"nodeName,omitempty"`
	// Events are the events of the pod that is running the workspace. This will only be filled if the pod is not running.
	// +optional
	Events []DevPodWorkspaceInstanceEvent `json:"events,omitempty"`
	// ContainerResources are the resources of the containers that are running the workspace
	// +optional
	ContainerResources []DevPodWorkspaceInstanceContainerResource `json:"containerResources,omitempty"`
	// ContainerMetrics are the metrics of the pod that is running the workspace
	// +optional
	ContainerMetrics []metricsv1beta1.ContainerMetrics `json:"containerMetrics,omitempty"`
}

type DevPodWorkspaceInstanceContainerResource struct {
	// Name is the name of the container
	// +optional
	Name string `json:"name,omitempty"`
	// Resources is the resources of the container
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type DevPodWorkspaceInstancePersistentVolumeClaimStatus struct {
	// phase represents the current phase of PersistentVolumeClaim.
	// +optional
	Phase corev1.PersistentVolumeClaimPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=PersistentVolumeClaimPhase"`
	// capacity represents the actual resources of the underlying volume.
	// +optional
	Capacity corev1.ResourceList `json:"capacity,omitempty" protobuf:"bytes,3,rep,name=capacity,casttype=ResourceList,castkey=ResourceName"`
	// conditions is the current Condition of persistent volume claim. If underlying persistent volume is being
	// resized then the Condition will be set to 'Resizing'.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []corev1.PersistentVolumeClaimCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,4,rep,name=conditions"`
	// Events are the events of the pod that is running the workspace. This will only be filled if the persistent volume claim is not bound.
	// +optional
	Events []DevPodWorkspaceInstanceEvent `json:"events,omitempty"`
}

type DevPodWorkspaceInstanceEvent struct {
	// This should be a short, machine understandable string that gives the reason
	// for the transition into the object's current status.
	// TODO: provide exact specification for format.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,3,opt,name=reason"`

	// A human-readable description of the status of this operation.
	// TODO: decide on maximum length.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`

	// The time at which the most recent occurrence of this event was recorded.
	// +optional
	LastTimestamp metav1.Time `json:"lastTimestamp,omitempty" protobuf:"bytes,7,opt,name=lastTimestamp"`

	// Type of this event (Normal, Warning), new types could be added in the future
	// +optional
	Type string `json:"type,omitempty" protobuf:"bytes,9,opt,name=type"`
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
