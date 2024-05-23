package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	InstanceScheduled             agentstoragev1.ConditionType = "Scheduled"
	InstanceTemplateSynced        agentstoragev1.ConditionType = "TemplateSynced"
	InstanceTemplateResolved      agentstoragev1.ConditionType = "TemplateResolved"
	InstanceSpaceSynced           agentstoragev1.ConditionType = "SpaceSynced"
	InstanceSpaceReady            agentstoragev1.ConditionType = "SpaceReady"
	InstanceVirtualClusterSynced  agentstoragev1.ConditionType = "VirtualClusterSynced"
	InstanceVirtualClusterReady   agentstoragev1.ConditionType = "VirtualClusterReady"
	InstanceProjectsSecretsSynced agentstoragev1.ConditionType = "ProjectSecretsSynced"

	InstanceVirtualClusterAppsAndObjectsSynced agentstoragev1.ConditionType = "VirtualClusterAppsAndObjectsSynced"

	// Workload VirtualCluster conditions

	InstanceWorkloadSpaceSynced               agentstoragev1.ConditionType = "WorkloadSpaceSynced"
	InstanceWorkloadSpaceReady                agentstoragev1.ConditionType = "WorkloadSpaceReady"
	InstanceWorkloadVirtualClusterSynced      agentstoragev1.ConditionType = "WorkloadVirtualClusterSynced"
	InstanceWorkloadVirtualClusterTokenSynced agentstoragev1.ConditionType = "WorkloadVirtualClusterTokenSynced"
	InstanceWorkloadVirtualClusterReady       agentstoragev1.ConditionType = "WorkloadVirtualClusterReady"
)

var VirtualClusterConditions = []agentstoragev1.ConditionType{
	InstanceScheduled,
	InstanceTemplateResolved,
	InstanceSpaceSynced,
	InstanceSpaceReady,
	InstanceVirtualClusterSynced,
	InstanceVirtualClusterReady,
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterInstance
// +k8s:openapi-gen=true
type VirtualClusterInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterInstanceSpec   `json:"spec,omitempty"`
	Status VirtualClusterInstanceStatus `json:"status,omitempty"`
}

func (a *VirtualClusterInstance) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *VirtualClusterInstance) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *VirtualClusterInstance) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *VirtualClusterInstance) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *VirtualClusterInstance) GetAccess() []Access {
	return a.Spec.Access
}

func (a *VirtualClusterInstance) SetAccess(access []Access) {
	a.Spec.Access = access
}

type VirtualClusterInstanceSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes a virtual cluster instance
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// TemplateRef holds the virtual cluster template reference
	// +optional
	TemplateRef *TemplateRef `json:"templateRef,omitempty"`

	// Template is the inline template to use for virtual cluster creation. This is mutually
	// exclusive with templateRef.
	// +optional
	Template *VirtualClusterTemplateDefinition `json:"template,omitempty"`

	// ClusterRef is the reference to the connected cluster holding
	// this virtual cluster
	// +optional
	ClusterRef VirtualClusterClusterRef `json:"clusterRef,omitempty"`

	// WorkloadClusterRef is the reference to the connected cluster holding
	// this virtual cluster's workloads.
	// +optional
	WorkloadClusterRef *VirtualClusterClusterRef `json:"workloadClusterRef,omitempty"`

	// Parameters are values to pass to the template.
	// The values should be encoded as YAML string where each parameter is represented as a top-level field key.
	// +optional
	Parameters string `json:"parameters,omitempty"`

	// ExtraAccessRules defines extra rules which users and teams should have which access to the virtual
	// cluster.
	// +optional
	ExtraAccessRules []agentstoragev1.InstanceAccessRule `json:"extraAccessRules,omitempty"`

	// Access to the virtual cluster object itself
	// +optional
	Access []Access `json:"access,omitempty"`

	// NetworkPeer specifies if the cluster is connected via tailscale.
	// When this is specified, the vCluster will not be scheduled to any connected cluster
	// and no templates will be applied to it.
	// +optional
	NetworkPeer bool `json:"networkPeer,omitempty"`
}

type VirtualClusterInstanceStatus struct {
	// Phase describes the current phase the virtual cluster instance is in
	// +optional
	Phase InstancePhase `json:"phase,omitempty"`

	// Reason describes the reason in machine-readable form why the cluster is in the current
	// phase
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes the reason in human-readable form why the cluster is in the current
	// phase
	// +optional
	Message string `json:"message,omitempty"`

	// Conditions holds several conditions the virtual cluster might be in
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`

	// VirtualClusterObjects are the objects that were applied within the virtual cluster itself
	// +optional
	VirtualClusterObjects *agentstoragev1.ObjectsStatus `json:"virtualClusterObjects,omitempty"`

	// SpaceObjects are the objects that were applied within the virtual cluster space
	// +optional
	SpaceObjects *agentstoragev1.ObjectsStatus `json:"spaceObjects,omitempty"`

	// WorkloadSpaceObjects are the objects that were applied within the virtual cluster workload space
	// +optional
	WorkloadSpaceObjects *agentstoragev1.ObjectsStatus `json:"workloadSpaceObjects,omitempty"`

	// VirtualCluster is the template rendered with all the parameters
	// +optional
	VirtualCluster *VirtualClusterTemplateDefinition `json:"virtualCluster,omitempty"`

	// IgnoreReconciliation tells the controller to ignore reconciliation for this instance -- this
	// is primarily used when migrating virtual cluster instances from project to project; this
	// prevents a situation where there are two virtual cluster instances representing the same
	// virtual cluster which could cause issues with concurrent reconciliations of the same object.
	// Once the virtual cluster instance has been cloned and placed into the new project, this
	// (the "old") virtual cluster instance can safely be deleted.
	IgnoreReconciliation bool `json:"ignoreReconciliation,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterInstanceList contains a list of VirtualClusterInstance objects
type VirtualClusterInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualClusterInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualClusterInstance{}, &VirtualClusterInstanceList{})
}
