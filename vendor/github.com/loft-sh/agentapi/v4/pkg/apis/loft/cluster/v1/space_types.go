package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SpacePodSecurityLabel = "policy.loft.sh/pod-security"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Space
// +k8s:openapi-gen=true
// +resource:path=spaces,rest=SpaceREST
type Space struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpaceSpec   `json:"spec,omitempty"`
	Status SpaceStatus `json:"status,omitempty"`
}

// SpaceSpec defines the desired state of Space
type SpaceSpec struct {
	// User is the owning user of the space
	// +optional
	User string `json:"user,omitempty"`

	// Team is the owning team of the space
	// +optional
	Team string `json:"team,omitempty"`

	// Objects are Kubernetes style yamls that should get deployed into the space
	// +optional
	Objects string `json:"objects,omitempty"`

	// Finalizers is an opaque list of values that must be empty to permanently remove object from storage.
	// More info: https://kubernetes.io/docs/tasks/administer-cluster/namespaces/
	// +optional
	Finalizers []corev1.FinalizerName `json:"finalizers,omitempty"`
}

// SpaceStatus defines the observed state of Space
type SpaceStatus struct {
	// Phase is the current lifecycle phase of the namespace.
	// More info: https://kubernetes.io/docs/tasks/administer-cluster/namespaces/
	// +optional
	Phase corev1.NamespacePhase `json:"phase,omitempty"`

	// SleepModeConfig is the sleep mode config of the space
	// +optional
	SleepModeConfig *SleepModeConfig `json:"sleepModeConfig,omitempty"`

	// Owner describes the owner of the space. This can be either empty (nil), be a team or
	// an loft user. If the space has an account that does not belong to an user / team in loft
	// this is empty
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// SpaceObjectsStatus describes the status of applying space objects.
	// +optional
	SpaceObjectsStatus *SpaceObjectsNamespaceStatus `json:"spaceObjectsStatus,omitempty"`

	// TemplateSyncStatus describes the template sync status
	// +optional
	TemplateSyncStatus *TemplateSyncStatus `json:"templateSyncStatus,omitempty"`
}

type TemplateSyncStatus struct {
	// Template is the json string of the template that was applied
	Template string `json:"template,omitempty"`

	// Phase indicates the current phase the template is in
	Phase string `json:"phase,omitempty"`
}

const (
	OutOfSyncPhase = "OutOfSync"
)

type SpaceConstraintNamespaceStatus struct {
	// SpaceConstraint are the applied space constraints
	SpaceConstraint string `json:"spaceConstraint,omitempty"`

	// User that was used to apply the space constraints
	User string `json:"user,omitempty"`

	// Team that was used to apply the space constraints
	Team string `json:"team,omitempty"`

	// ObservedGeneration of the space constraint
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase the namespace is in
	Phase string `json:"phase,omitempty"`

	// Reason why this namespace is in the current phase
	Reason string `json:"reason,omitempty"`

	// Message is the human-readable message why this space is in this phase
	Message string `json:"message,omitempty"`

	// AppliedClusterRole is the cluster role that was bound to this namespace
	AppliedClusterRole *string `json:"appliedClusterRole,omitempty"`

	// AppliedMetadata is the metadata that was applied on the space
	AppliedMetadata AppliedMetadata `json:"appliedMetadata,omitempty"`

	// AppliedObjects are the objects that were applied on this namespace by the space constraint
	AppliedObjects []AppliedObject `json:"appliedObjects,omitempty"`
}

type SpaceObjectsNamespaceStatus struct {
	// Phase the namespace is in
	Phase string `json:"phase,omitempty"`

	// Reason why this namespace is in the current phase
	Reason string `json:"reason,omitempty"`

	// Message is the human-readable message why this space is in this phase
	Message string `json:"message,omitempty"`

	// AppliedObjects are the objects that were applied on this namespace by the space spec objects
	AppliedObjects []AppliedObject `json:"appliedObjects,omitempty"`
}

type AppliedMetadata struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type AppliedObject struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name,omitempty"`
}

const (
	PhaseSynced = "Synced"
	PhaseError  = "Error"
)
