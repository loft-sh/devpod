package v1

import (
	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskSpec   `json:"spec,omitempty"`
	Status TaskStatus `json:"status,omitempty"`
}

// GetConditions returns the set of conditions for this object.
func (a *Task) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

// SetConditions sets the conditions on this object.
func (a *Task) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *Task) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *Task) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *Task) GetAccess() []Access {
	return a.Spec.Access
}

func (a *Task) SetAccess(access []Access) {
	a.Spec.Access = access
}

type TaskSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`

	// Scope defines the scope of the access key.
	// +optional
	Scope *AccessKeyScope `json:"scope,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Target where this task should get executed
	// +optional
	Target Target `json:"target,omitempty"`

	// Task defines the task to execute
	// +optional
	Task TaskDefinition `json:"task,omitempty"`
}

type TaskDefinition struct {
	// AppTask is an app task
	// +optional
	AppTask *AppTask `json:"appTask,omitempty"`

	// HelmTask executes a helm command
	// +optional
	HelmTask *HelmTask `json:"helm,omitempty"`
}

type AppTask struct {
	// Type is the task type. Defaults to Upgrade
	// +optional
	Type HelmTaskType `json:"type,omitempty"`

	// RollbackRevision is the revision to rollback to
	// +optional
	RollbackRevision string `json:"rollbackRevision,omitempty"`

	// AppReference is the reference to the app to deploy
	// +optional
	AppReference AppReference `json:"appReference,omitempty"`
}

type HelmTask struct {
	// Release holds the release information
	// +optional
	Release HelmTaskRelease `json:"release,omitempty"`

	// Type is the task type. Defaults to Upgrade
	// +optional
	Type HelmTaskType `json:"type,omitempty"`

	// RollbackRevision is the revision to rollback to
	// +optional
	RollbackRevision string `json:"rollbackRevision,omitempty"`
}

type HelmTaskRelease struct {
	// Name is the name of the release
	Name string `json:"name,omitempty"`

	// Namespace of the release, if empty will use the target namespace
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Config is the helm config to use to deploy the release
	// +optional
	Config clusterv1.HelmReleaseConfig `json:"config,omitempty"`

	// =======================
	// DEPRECATED FIELDS BELOW
	// =======================

	// Labels are additional labels for the helm release.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

type StreamContainer struct {
	// Label selector for pods. The newest matching pod will be used to stream logs from
	// +optional
	Selector metav1.LabelSelector `json:"selector" protobuf:"bytes,2,opt,name=selector"`

	// Container is the container name to use
	// +optional
	Container string `json:"container,omitempty"`
}

type TaskStatus struct {
	// Started determines if the task was started
	// +optional
	Started bool `json:"started,omitempty"`

	// Conditions holds several conditions the virtual cluster might be in
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`

	// PodPhase describes the phase this task is in
	// +optional
	PodPhase corev1.PodPhase `json:"podPhase,omitempty"`

	// ObservedGeneration is the latest generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// DEPRECATED: This is not set anymore after migrating to runners
	// ContainerState describes the container state of the task
	// +optional
	ContainerState *corev1.ContainerStatus `json:"containerState,omitempty"`
}

// Common ConditionTypes used by Cluster API objects.
const (
	// TaskStartedCondition defines the task started condition type that summarizes the operational state of the virtual cluster API object.
	TaskStartedCondition agentstoragev1.ConditionType = "TaskStarted"
)

// HelmTaskType describes the type of a task
type HelmTaskType string

// These are the valid admin account types
const (
	HelmTaskTypeInstall  HelmTaskType = "Install"
	HelmTaskTypeUpgrade  HelmTaskType = "Upgrade"
	HelmTaskTypeDelete   HelmTaskType = "Delete"
	HelmTaskTypeRollback HelmTaskType = "Rollback"
)

type Target struct {
	// SpaceInstance defines a space instance as target
	// +optional
	SpaceInstance *TargetInstance `json:"spaceInstance,omitempty"`

	// VirtualClusterInstance defines a virtual cluster instance as target
	// +optional
	VirtualClusterInstance *TargetInstance `json:"virtualClusterInstance,omitempty"`

	// Cluster defines a connected cluster as target
	// +optional
	Cluster *TargetCluster `json:"cluster,omitempty"`
}

type TargetInstance struct {
	// Name is the name of the instance
	// +optional
	Name string `json:"name,omitempty"`

	// Project where the instance is in
	// +optional
	Project string `json:"project,omitempty"`
}

type TargetCluster struct {
	// Cluster is the cluster where the task should get executed
	// +optional
	Cluster string `json:"cluster,omitempty"`

	// Namespace is the namespace where the task should get executed
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

type TargetVirtualCluster struct {
	// Cluster is the cluster where the virtual cluster lies
	// +optional
	Cluster string `json:"cluster,omitempty"`

	// Namespace is the namespace where the virtual cluster is located
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Name of the virtual cluster
	// +optional
	Name string `json:"name,omitempty"`
}

type UserOrTeamEntity struct {
	// User describes an user
	// +optional
	User *EntityInfo `json:"user,omitempty"`

	// Team describes a team
	// +optional
	Team *EntityInfo `json:"team,omitempty"`
}

type EntityInfo struct {
	// Name is the kubernetes name of the object
	Name string `json:"name,omitempty"`

	// The display name shown in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Icon is the icon of the user / team
	// +optional
	Icon string `json:"icon,omitempty"`

	// The username that is used to login
	// +optional
	Username string `json:"username,omitempty"`

	// The users email address
	// +optional
	Email string `json:"email,omitempty"`

	// The user subject
	// +optional
	Subject string `json:"subject,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TaskList contains a list of Task
type TaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Task `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Task{}, &TaskList{})
}
