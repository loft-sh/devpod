package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var RunnerConditions = []agentstoragev1.ConditionType{
	RunnerDeployed,
}

const (
	RunnerDeployed agentstoragev1.ConditionType = "Deployed"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Runner holds the runner information
// +k8s:openapi-gen=true
type Runner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunnerSpec   `json:"spec,omitempty"`
	Status RunnerStatus `json:"status,omitempty"`
}

func (a *Runner) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *Runner) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *Runner) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *Runner) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *Runner) GetAccess() []Access {
	return a.Spec.Access
}

func (a *Runner) SetAccess(access []Access) {
	a.Spec.Access = access
}

type RunnerSpec struct {
	// The display name shown in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes a cluster access object
	// +optional
	Description string `json:"description,omitempty"`

	// NetworkPeerName is the network peer name used to connect directly to the runner
	// +optional
	NetworkPeerName string `json:"networkPeerName,omitempty"`

	// If ClusterRef is defined, Loft will schedule the runner on the given
	// cluster.
	// +optional
	ClusterRef *RunnerClusterRef `json:"clusterRef,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// If unusable is true, no DevPod workspaces can be scheduled on this runner.
	// +optional
	Unusable bool `json:"unusable,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`
}

type RunnerClusterRef struct {
	// Cluster is the connected cluster the space will be created in
	// +optional
	Cluster string `json:"cluster,omitempty"`

	// Namespace is the namespace inside the connected cluster holding the space
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// PersistentVolumeClaimTemplate holds additional options for the persistent volume claim
	// +optional
	PersistentVolumeClaimTemplate *RunnerPersistentVolumeClaimTemplate `json:"persistentVolumeClaimTemplate,omitempty"`

	// PodTemplate holds additional options for the runner pod
	// +optional
	PodTemplate *RunnerPodTemplate `json:"podTemplate,omitempty"`
}

type RunnerPodTemplate struct {
	// Metadata holds the template metadata
	// +optional
	Metadata TemplateMetadata `json:"metadata,omitempty"`

	// Spec holds the template spec
	// +optional
	Spec RunnerPodTemplateSpec `json:"spec,omitempty"`
}

type RunnerPodTemplateSpec struct {
	// Runner pod image to use other than default
	// +optional
	Image string `json:"image,omitempty"`

	// Resources requirements
	// +optional
	Resources corev1.ResourceRequirements `json:"resource,omitempty"`

	// List of sources to populate environment variables in the container.
	// The keys defined within a source must be a C_IDENTIFIER. All invalid keys
	// will be reported as an event when the container is starting. When a key exists in multiple
	// sources, the value associated with the last source will take precedence.
	// Values defined by an Env with a duplicate key will take precedence.
	// Cannot be updated.
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// List of environment variables to set in the container.
	// Cannot be updated.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Set the NodeSelector for the Runner Pod
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Set the Affinity for the Runner Pod
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Set the Tolerations for the Runner Pod
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Set Volume Mounts for the Runner Pod
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// Set Volumes for the Runner Pod
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// Set up Init Containers for the Runner
	// +optional
	InitContainers []corev1.Container `json:"initContainers,omitempty"`

	// Set host aliases for the Runner Pod
	// +optional
	HostAliases []corev1.HostAlias `json:"hostAliases,omitempty"`
}

type RunnerPersistentVolumeClaimTemplate struct {
	// Metadata holds the template metadata
	// +optional
	Metadata TemplateMetadata `json:"metadata,omitempty"`

	// Spec holds the template spec
	// +optional
	Spec RunnerPersistentVolumeClaimTemplateSpec `json:"spec,omitempty"`
}

type RunnerPersistentVolumeClaimTemplateSpec struct {
	// accessModes contains the desired access modes the volume should have.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1
	// +optional
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`

	// storageClassName is the name of the StorageClass required by the claim.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// storageSize is the size of the storage to reserve for the pvc
	// +optional
	StorageSize string `json:"storageSize,omitempty"`
}

type RunnerStatus struct {
	// Phase describes the current phase the space instance is in
	// +optional
	Phase RunnerStatusPhase `json:"phase,omitempty"`

	// Reason describes the reason in machine-readable form
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes the reason in human-readable form
	// +optional
	Message string `json:"message,omitempty"`

	// Conditions holds several conditions the virtual cluster might be in
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`
}

// RunnerStatusPhase describes the phase of a cluster
type RunnerStatusPhase string

// These are the valid admin account types
const (
	RunnerStatusPhaseInitializing RunnerStatusPhase = ""
	RunnerStatusPhaseInitialized  RunnerStatusPhase = "Initialized"
	RunnerStatusPhaseFailed       RunnerStatusPhase = "Failed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RunnerList contains a list of Runner
type RunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Runner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Runner{}, &RunnerList{})
}
