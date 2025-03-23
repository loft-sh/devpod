package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

var (
	DevPodWorkspaceConditions = []agentstoragev1.ConditionType{
		InstanceScheduled,
		InstanceTemplateResolved,
	}

	// DevPodWorkspaceIDLabel holds the actual workspace id of the devpod workspace
	DevPodWorkspaceIDLabel = "loft.sh/workspace-id"

	// DevPodWorkspaceUIDLabel holds the actual workspace uid of the devpod workspace
	DevPodWorkspaceUIDLabel = "loft.sh/workspace-uid"

	// DevPodKubernetesProviderWorkspaceUIDLabel holds the actual workspace uid of the devpod workspace on resources
	// created by the DevPod Kubernetes provider.
	DevPodKubernetesProviderWorkspaceUIDLabel = "devpod.sh/workspace-uid"

	// DevPodWorkspacePictureAnnotation holds the workspace picture url of the devpod workspace
	DevPodWorkspacePictureAnnotation = "loft.sh/workspace-picture"

	// DevPodWorkspaceSourceAnnotation holds the workspace source of the devpod workspace
	DevPodWorkspaceSourceAnnotation = "loft.sh/workspace-source"

	// DevPodClientsAnnotation holds the active clients for a workspace networpeer
	DevPodClientsAnnotation = "loft.sh/devpod-clients"
)

var (
	DevPodPlatformOptions = "DEVPOD_PLATFORM_OPTIONS"

	DevPodFlagsUp     = "DEVPOD_FLAGS_UP"
	DevPodFlagsDelete = "DEVPOD_FLAGS_DELETE"
	DevPodFlagsStop   = "DEVPOD_FLAGS_STOP"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DevPodWorkspaceInstance
// +k8s:openapi-gen=true
type DevPodWorkspaceInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevPodWorkspaceInstanceSpec   `json:"spec,omitempty"`
	Status DevPodWorkspaceInstanceStatus `json:"status,omitempty"`
}

func (a *DevPodWorkspaceInstance) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *DevPodWorkspaceInstance) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *DevPodWorkspaceInstance) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *DevPodWorkspaceInstance) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *DevPodWorkspaceInstance) GetAccess() []Access {
	return a.Spec.Access
}

func (a *DevPodWorkspaceInstance) SetAccess(access []Access) {
	a.Spec.Access = access
}

type DevPodWorkspaceInstanceSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes a DevPod machine instance
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// PresetRef holds the DevPodWorkspacePreset template reference
	// +optional
	PresetRef *PresetRef `json:"presetRef,omitempty"`

	// TemplateRef holds the DevPod machine template reference
	// +optional
	TemplateRef *TemplateRef `json:"templateRef,omitempty"`

	// EnvironmentRef is the reference to DevPodEnvironmentTemplate that should be used
	// +optional
	EnvironmentRef *EnvironmentRef `json:"environmentRef,omitempty"`

	// Template is the inline template to use for DevPod machine creation. This is mutually
	// exclusive with templateRef.
	// +optional
	Template *DevPodWorkspaceTemplateDefinition `json:"template,omitempty"`

	// Target is the reference to the cluster holding this workspace
	// +optional
	Target WorkspaceTarget `json:"target,omitempty"`

	// Deprecated: Use TargetRef instead
	// RunnerRef is the reference to the runner holding this workspace
	// +optional
	RunnerRef RunnerRef `json:"runnerRef,omitempty"`

	// Parameters are values to pass to the template.
	// The values should be encoded as YAML string where each parameter is represented as a top-level field key.
	// +optional
	Parameters string `json:"parameters,omitempty"`

	// Access to the DevPod machine instance object itself
	// +optional
	Access []Access `json:"access,omitempty"`

	// PreventWakeUpOnConnection is used to prevent workspace that uses sleep mode from waking up on incomming ssh connection.
	// +optional
	PreventWakeUpOnConnection bool `json:"preventWakeUpOnConnection,omitempty"`
}

type PresetRef struct {
	// Name is the name of DevPodWorkspacePreset
	Name string `json:"name"`

	// Version holds the preset version to use. Version is expected to
	// be in semantic versioning format. Alternatively, you can also exchange
	// major, minor or patch with an 'x' to tell Loft to automatically select
	// the latest major, minor or patch version.
	// +optional
	Version string `json:"version,omitempty"`
}

type WorkspaceTarget struct {
	// Cluster is the reference to the cluster holding this workspace
	// +optional
	Cluster *WorkspaceTargetName `json:"cluster,omitempty"`

	// Cluster is the reference to the virtual cluster holding this workspace
	// +optional
	VirtualCluster *WorkspaceTargetName `json:"virtualCluster,omitempty"`
}

type WorkspaceResolvedTarget struct {
	// Cluster is the reference to the cluster holding this workspace
	// +optional
	Cluster *WorkspaceTargetNamespace `json:"cluster,omitempty"`

	// Cluster is the reference to the virtual cluster holding this workspace
	// +optional
	VirtualCluster *WorkspaceTargetNamespace `json:"virtualCluster,omitempty"`

	// Space is the reference to the space holding this workspace
	// +optional
	Space *WorkspaceTargetName `json:"space,omitempty"`
}

func (w WorkspaceResolvedTarget) Empty() bool {
	return w == WorkspaceResolvedTarget{}
}

type WorkspaceTargetName struct {
	// Name is the name of the target
	Name string `json:"name"`
}

type WorkspaceTargetNamespace struct {
	// Name is the name of the object
	Name string `json:"name"`

	// Namespace is the namespace within the cluster.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

type RunnerRef struct {
	// Runner is the connected runner the workspace will be created in
	// +optional
	Runner string `json:"runner,omitempty"`
}

type EnvironmentRef struct {
	// Name is the name of DevPodEnvironmentTemplate this references
	Name string `json:"name"`

	// Version is the version of DevPodEnvironmentTemplate this references
	// +optional
	Version string `json:"version,omitempty"`
}

type DevPodWorkspaceInstanceStatus struct {
	// ResolvedTarget is the resolved target of the workspace
	// +optional
	ResolvedTarget WorkspaceResolvedTarget `json:"resolvedTarget,omitempty"`

	// LastWorkspaceStatus is the last workspace status reported by the runner.
	// +optional
	LastWorkspaceStatus WorkspaceStatus `json:"lastWorkspaceStatus,omitempty"`

	// Phase describes the current phase the DevPod machine instance is in
	// +optional
	Phase InstancePhase `json:"phase,omitempty"`

	// Reason describes the reason in machine-readable form why the cluster is in the current
	// phase
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes the reason in human-readable form why the DevPod machine is in the current
	// phase
	// +optional
	Message string `json:"message,omitempty"`

	// Conditions holds several conditions the DevPod machine might be in
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`

	// Instance is the template rendered with all the parameters
	// +optional
	Instance *DevPodWorkspaceTemplateDefinition `json:"instance,omitempty"`

	// IgnoreReconciliation ignores reconciliation for this object
	// +optional
	IgnoreReconciliation bool `json:"ignoreReconciliation,omitempty"`

	// Kubernetes is the status of the workspace on kubernetes
	// +optional
	Kubernetes *DevPodWorkspaceInstanceKubernetesStatus `json:"kubernetes,omitempty"`
}

type DevPodWorkspaceInstanceKubernetesStatus struct {
	// Last time the condition transitioned from one status to another.
	// +required
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

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

type WorkspaceStatusResult struct {
	ID       string `json:"id,omitempty"`
	Context  string `json:"context,omitempty"`
	Provider string `json:"provider,omitempty"`
	State    string `json:"state,omitempty"`
}

var AllowedWorkspaceStatus = []WorkspaceStatus{
	WorkspaceStatusNotFound,
	WorkspaceStatusStopped,
	WorkspaceStatusBusy,
	WorkspaceStatusRunning,
}

type WorkspaceStatus string

var (
	WorkspaceStatusNotFound WorkspaceStatus = "NotFound"
	WorkspaceStatusStopped  WorkspaceStatus = "Stopped"
	WorkspaceStatusBusy     WorkspaceStatus = "Busy"
	WorkspaceStatusRunning  WorkspaceStatus = "Running"
)

type DevPodCommandStopOptions struct{}

type DevPodCommandDeleteOptions struct {
	IgnoreNotFound bool   `json:"ignoreNotFound,omitempty"`
	Force          bool   `json:"force,omitempty"`
	GracePeriod    string `json:"gracePeriod,omitempty"`
}

type DevPodCommandStatusOptions struct {
	ContainerStatus bool `json:"containerStatus,omitempty"`
}

type DevPodCommandUpOptions struct {
	// up options
	ID                   string   `json:"id,omitempty"`
	Source               string   `json:"source,omitempty"`
	IDE                  string   `json:"ide,omitempty"`
	IDEOptions           []string `json:"ideOptions,omitempty"`
	PrebuildRepositories []string `json:"prebuildRepositories,omitempty"`
	DevContainerPath     string   `json:"devContainerPath,omitempty"`
	WorkspaceEnv         []string `json:"workspaceEnv,omitempty"`
	Recreate             bool     `json:"recreate,omitempty"`
	Proxy                bool     `json:"proxy,omitempty"`
	DisableDaemon        bool     `json:"disableDaemon,omitempty"`
	DaemonInterval       string   `json:"daemonInterval,omitempty"`

	// build options
	Repository string   `json:"repository,omitempty"`
	SkipPush   bool     `json:"skipPush,omitempty"`
	Platform   []string `json:"platform,omitempty"`

	// TESTING
	ForceBuild            bool `json:"forceBuild,omitempty"`
	ForceInternalBuildKit bool `json:"forceInternalBuildKit,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DevPodWorkspaceInstanceList contains a list of DevPodWorkspaceInstance objects
type DevPodWorkspaceInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DevPodWorkspaceInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DevPodWorkspaceInstance{}, &DevPodWorkspaceInstanceList{})
}
