package v1

import (
	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	InstanceScheduled              agentstoragev1.ConditionType = "Scheduled"
	InstanceTemplateSynced         agentstoragev1.ConditionType = "TemplateSynced"
	InstanceTemplateResolved       agentstoragev1.ConditionType = "TemplateResolved"
	InstanceSpaceSynced            agentstoragev1.ConditionType = "SpaceSynced"
	InstanceSpaceReady             agentstoragev1.ConditionType = "SpaceReady"
	InstanceVirtualClusterDeployed agentstoragev1.ConditionType = "VirtualClusterDeployed"
	InstanceVirtualClusterSynced   agentstoragev1.ConditionType = "VirtualClusterSynced"
	InstanceVirtualClusterReady    agentstoragev1.ConditionType = "VirtualClusterReady"
	InstanceVirtualClusterOnline   agentstoragev1.ConditionType = "VirtualClusterOnline"
	InstanceProjectsSecretsSynced  agentstoragev1.ConditionType = "ProjectSecretsSynced"

	InstanceVirtualClusterAppsAndObjectsSynced agentstoragev1.ConditionType = "VirtualClusterAppsAndObjectsSynced"
)

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

	// Parameters are values to pass to the template.
	// The values should be encoded as YAML string where each parameter is represented as a top-level field key.
	// +optional
	Parameters string `json:"parameters,omitempty"`

	// ExtraAccessRules defines extra rules which users and teams should have which access to the virtual
	// cluster.
	// +optional
	ExtraAccessRules []InstanceAccessRule `json:"extraAccessRules,omitempty"`

	// Access to the virtual cluster object itself
	// +optional
	Access []Access `json:"access,omitempty"`

	// NetworkPeer specifies if the cluster is connected via tailscale.
	// +optional
	NetworkPeer bool `json:"networkPeer,omitempty"`

	// External specifies if the virtual cluster is managed by the platform agent or externally.
	// +optional
	External bool `json:"external,omitempty"`
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

	// ServiceUID is the service uid of the virtual cluster to uniquely identify it.
	// +optional
	ServiceUID string `json:"serviceUID,omitempty"`

	// DeployHash is the hash of the last deployed values.
	// +optional
	DeployHash string `json:"deployHash,omitempty"`

	// Conditions holds several conditions the virtual cluster might be in
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`

	// VirtualClusterObjects are the objects that were applied within the virtual cluster itself
	// +optional
	VirtualClusterObjects *ObjectsStatus `json:"virtualClusterObjects,omitempty"`

	// SpaceObjects are the objects that were applied within the virtual cluster space
	// +optional
	SpaceObjects *ObjectsStatus `json:"spaceObjects,omitempty"`

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

// VirtualClusterCommonSpec holds common attributes for virtual clusters and virtual cluster templates
type VirtualClusterCommonSpec struct {
	// Apps specifies the apps that should get deployed by this template
	// +optional
	Apps []AppReference `json:"apps,omitempty"`

	// Charts are helm charts that should get deployed
	// +optional
	Charts []TemplateHelmChart `json:"charts,omitempty"`

	// Objects are Kubernetes style yamls that should get deployed into the virtual cluster
	// +optional
	Objects string `json:"objects,omitempty"`

	// Access defines the access of users and teams to the virtual cluster.
	// +optional
	Access *InstanceAccess `json:"access,omitempty"`

	// Pro defines the pro settings for the virtual cluster
	// +optional
	Pro VirtualClusterProSpec `json:"pro,omitempty"`

	// HelmRelease is the helm release configuration for the virtual cluster.
	// +optional
	HelmRelease VirtualClusterHelmRelease `json:"helmRelease,omitempty"`

	// AccessPoint defines settings to expose the virtual cluster directly via an ingress rather than
	// through the (default) Loft proxy
	// +optional
	AccessPoint VirtualClusterAccessPoint `json:"accessPoint,omitempty"`

	// ForwardToken signals the proxy to pass through the used token to the virtual Kubernetes
	// api server and do a TokenReview there.
	// +optional
	ForwardToken bool `json:"forwardToken,omitempty"`
}

type VirtualClusterProSpec struct {
	// Enabled defines if the virtual cluster is a pro cluster or not
	// +optional
	Enabled bool `json:"enabled,omitempty"`
}

type VirtualClusterAccessPoint struct {
	// Ingress defines virtual cluster access via ingress
	// +optional
	Ingress VirtualClusterAccessPointIngressSpec `json:"ingress,omitempty"`
}

type VirtualClusterAccessPointIngressSpec struct {
	// Enabled defines if the virtual cluster access point (via ingress) is enabled or not; requires
	// the connected cluster to have the `loft.sh/ingress-suffix` annotation set to define the domain
	// name suffix used for the ingress.
	Enabled bool `json:"enabled,omitempty"`
}

type TemplateHelmChart struct {
	clusterv1.Chart `json:",inline"`

	// ReleaseName is the preferred release name of the app
	// +optional
	ReleaseName string `json:"releaseName,omitempty"`

	// ReleaseNamespace is the preferred release namespace of the app
	// +optional
	ReleaseNamespace string `json:"releaseNamespace,omitempty"`

	// Values are the values that should get passed to the chart
	// +optional
	Values string `json:"values,omitempty"`

	// Wait determines if Loft should wait during deploy for the app to become ready
	// +optional
	Wait bool `json:"wait,omitempty"`

	// Timeout is the time to wait for any individual Kubernetes operation (like Jobs for hooks) (default 5m0s)
	// +optional
	Timeout string `json:"timeout,omitempty"`
}

type InstanceAccess struct {
	// Specifies which cluster role should get applied to users or teams that do not
	// match a rule below.
	// +optional
	DefaultClusterRole string `json:"defaultClusterRole,omitempty"`

	// Rules defines which users and teams should have which access to the virtual
	// cluster. If no rule matches an authenticated incoming user, the user will get cluster admin
	// access.
	// +optional
	Rules []InstanceAccessRule `json:"rules,omitempty"`
}

type InstanceAccessRule struct {
	// Users this rule matches. * means all users.
	// +optional
	Users []string `json:"users,omitempty"`

	// Teams that this rule matches.
	// +optional
	Teams []string `json:"teams,omitempty"`

	// ClusterRole is the cluster role that should be assigned to the
	// +optional
	ClusterRole string `json:"clusterRole,omitempty"`
}

type AppReference struct {
	// Name of the target app
	// +optional
	Name string `json:"name,omitempty"`

	// Namespace specifies in which target namespace the app should
	// get deployed in
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// ReleaseName is the name of the app release
	// +optional
	ReleaseName string `json:"releaseName,omitempty"`

	// Version of the app
	// +optional
	Version string `json:"version,omitempty"`

	// Parameters to use for the app
	// +optional
	Parameters string `json:"parameters,omitempty"`
}

type VirtualClusterHelmRelease struct {
	// infos about what chart to deploy
	// +optional
	Chart VirtualClusterHelmChart `json:"chart,omitempty"`

	// the values for the given chart
	// +optional
	Values string `json:"values,omitempty"`
}

type VirtualClusterHelmChart struct {
	// the name of the helm chart
	// +optional
	Name string `json:"name,omitempty"`

	// the repo of the helm chart
	// +optional
	Repo string `json:"repo,omitempty"`

	// The username that is required for this repository
	// +optional
	Username string `json:"username,omitempty"`

	// The password that is required for this repository
	// +optional
	Password string `json:"password,omitempty"`

	// the version of the helm chart to use
	// +optional
	Version string `json:"version,omitempty"`
}

type PodSelector struct {
	// A label selector to select the virtual cluster pod to route
	// incoming requests to.
	// +optional
	Selector metav1.LabelSelector `json:"podSelector,omitempty"`

	// The port of the pod to route to
	// +optional
	Port *int `json:"port,omitempty"`
}

// VirtualClusterStatus holds the status of a virtual cluster
type VirtualClusterStatus struct {
	// Phase describes the current phase the virtual cluster is in
	// +optional
	Phase VirtualClusterPhase `json:"phase,omitempty"`

	// Reason describes the reason in machine readable form why the cluster is in the current
	// phase
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes the reason in human readable form why the cluster is in the current
	// phase
	// +optional
	Message string `json:"message,omitempty"`

	// ControlPlaneReady defines if the virtual cluster control plane is ready.
	// +optional
	ControlPlaneReady bool `json:"controlPlaneReady,omitempty"`

	// Conditions holds several conditions the virtual cluster might be in
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`

	// ObservedGeneration is the latest generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// VirtualClusterObjects are the objects that were applied within the virtual cluster itself
	// +optional
	VirtualClusterObjects *ObjectsStatus `json:"virtualClusterObjects,omitempty"`

	// DeployHash saves the latest applied chart hash
	// +optional
	DeployHash string `json:"deployHash,omitempty"`

	// MultiNamespace indicates if this is a multinamespace enabled virtual cluster
	MultiNamespace bool `json:"multiNamespace,omitempty"`

	// DEPRECATED: do not use anymore
	// the status of the helm release that was used to deploy the virtual cluster
	// +optional
	HelmRelease *VirtualClusterHelmReleaseStatus `json:"helmRelease,omitempty"`
}

type ObjectsStatus struct {
	// LastAppliedObjects holds the status for the objects that were applied
	// +optional
	LastAppliedObjects string `json:"lastAppliedObjects,omitempty"`

	// Charts are the charts that were applied
	// +optional
	Charts []ChartStatus `json:"charts,omitempty"`

	// Apps are the apps that were applied
	// +optional
	Apps []AppReference `json:"apps,omitempty"`
}

type ChartStatus struct {
	// Name of the chart that was applied
	// +optional
	Name string `json:"name,omitempty"`

	// Namespace of the chart that was applied
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// LastAppliedChartConfigHash is the last applied configuration
	// +optional
	LastAppliedChartConfigHash string `json:"lastAppliedChartConfigHash,omitempty"`
}

type VirtualClusterHelmReleaseStatus struct {
	// +optional
	Phase string `json:"phase,omitempty"`

	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`

	// the release that was deployed
	// +optional
	Release VirtualClusterHelmRelease `json:"release,omitempty"`
}

// VirtualClusterPhase describes the phase of a virtual cluster
type VirtualClusterPhase string

// These are the valid admin account types
const (
	VirtualClusterUnknown  VirtualClusterPhase = ""
	VirtualClusterPending  VirtualClusterPhase = "Pending"
	VirtualClusterDeployed VirtualClusterPhase = "Deployed"
	VirtualClusterFailed   VirtualClusterPhase = "Failed"
)

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
