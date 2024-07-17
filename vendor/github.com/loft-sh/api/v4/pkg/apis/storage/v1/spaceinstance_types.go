package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	SpaceConditions = []agentstoragev1.ConditionType{
		InstanceScheduled,
		InstanceTemplateResolved,
		InstanceSpaceSynced,
		InstanceSpaceReady,
	}
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SpaceInstance
// +k8s:openapi-gen=true
type SpaceInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpaceInstanceSpec   `json:"spec,omitempty"`
	Status SpaceInstanceStatus `json:"status,omitempty"`
}

func (a *SpaceInstance) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *SpaceInstance) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *SpaceInstance) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *SpaceInstance) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *SpaceInstance) GetAccess() []Access {
	return a.Spec.Access
}

func (a *SpaceInstance) SetAccess(access []Access) {
	a.Spec.Access = access
}

type SpaceInstanceSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes a space instance
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// TemplateRef holds the space template reference
	// +optional
	TemplateRef *TemplateRef `json:"templateRef,omitempty"`

	// Template is the inline template to use for space creation. This is mutually
	// exclusive with templateRef.
	// +optional
	Template *SpaceTemplateDefinition `json:"template,omitempty"`

	// ClusterRef is the reference to the connected cluster holding
	// this space
	// +optional
	ClusterRef ClusterRef `json:"clusterRef,omitempty"`

	// Parameters are values to pass to the template.
	// The values should be encoded as YAML string where each parameter is represented as a top-level field key.
	// +optional
	Parameters string `json:"parameters,omitempty"`

	// ExtraAccessRules defines extra rules which users and teams should have which access to the virtual
	// cluster.
	// +optional
	ExtraAccessRules []InstanceAccessRule `json:"extraAccessRules,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`
}

type ClusterRef struct {
	// Cluster is the connected cluster the space will be created in
	// +optional
	Cluster string `json:"cluster,omitempty"`

	// Namespace is the namespace inside the connected cluster holding the space
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

type VirtualClusterClusterRef struct {
	ClusterRef `json:",inline"`

	// VirtualCluster is the name of the virtual cluster inside the namespace
	// +optional
	VirtualCluster string `json:"virtualCluster,omitempty"`
}

type TemplateRef struct {
	// Name holds the name of the template to reference.
	// +optional
	Name string `json:"name,omitempty"`

	// Version holds the template version to use. Version is expected to
	// be in semantic versioning format. Alternatively, you can also exchange
	// major, minor or patch with an 'x' to tell Loft to automatically select
	// the latest major, minor or patch version.
	// +optional
	Version string `json:"version,omitempty"`

	// SyncOnce tells the controller to sync the instance once with the template.
	// This is useful if you want to sync an instance after a template was changed.
	// To automatically sync an instance with a template, use 'x.x.x' as version
	// instead.
	// +optional
	SyncOnce bool `json:"syncOnce,omitempty"`
}

type SpaceInstanceStatus struct {
	// Phase describes the current phase the space instance is in
	// +optional
	Phase InstancePhase `json:"phase,omitempty"`

	// Reason describes the reason in machine-readable form
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes the reason in human-readable form
	// +optional
	Message string `json:"message,omitempty"`

	// Conditions holds several conditions the virtual cluster might be in
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`

	// SpaceObjects are the objects that were applied within the virtual cluster space
	// +optional
	SpaceObjects *ObjectsStatus `json:"spaceObjects,omitempty"`

	// Space is the template rendered with all the parameters
	// +optional
	Space *SpaceTemplateDefinition `json:"space,omitempty"`

	// IgnoreReconciliation tells the controller to ignore reconciliation for this instance -- this
	// is primarily used when migrating virtual cluster instances from project to project; this
	// prevents a situation where there are two virtual cluster instances representing the same
	// virtual cluster which could cause issues with concurrent reconciliations of the same object.
	// Once the virtual cluster instance has been cloned and placed into the new project, this
	// (the "old") virtual cluster instance can safely be deleted.
	IgnoreReconciliation bool `json:"ignoreReconciliation,omitempty"`
}

type InstanceDeployedAppStatus struct {
	// Name of the app that should get deployed
	// +optional
	Name string `json:"name,omitempty"`

	// Namespace specifies in which target namespace the app should
	// get deployed in. Only used for virtual cluster apps.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// ReleaseName of the target app
	// +optional
	ReleaseName string `json:"releaseName,omitempty"`

	// Version of the app that should get deployed
	// +optional
	Version string `json:"version,omitempty"`

	// Phase describes the current phase the app deployment is in
	// +optional
	Phase InstanceDeployedAppPhase `json:"phase,omitempty"`

	// Reason describes the reason in machine-readable form
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes the reason in human-readable form
	// +optional
	Message string `json:"message,omitempty"`
}

type InstanceDeployedAppPhase string

var (
	InstanceDeployedAppDeployed = "Deployed"
	InstanceDeployedAppFailed   = "Failed"
)

type InstancePhase string

var (
	InstanceReady    InstancePhase = "Ready"
	InstanceSleeping InstancePhase = "Sleeping"
	InstanceFailed   InstancePhase = "Failed"
	InstancePending  InstancePhase = "Pending"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SpaceInstanceList contains a list of SpaceInstance objects
type SpaceInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SpaceInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SpaceInstance{}, &SpaceInstanceList{})
}
