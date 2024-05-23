package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualCluster holds the information
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type VirtualCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterSpec   `json:"spec,omitempty"`
	Status VirtualClusterStatus `json:"status,omitempty"`
}

// GetConditions returns the set of conditions for this object.
func (in *VirtualCluster) GetConditions() Conditions {
	return in.Status.Conditions
}

// SetConditions sets the conditions on this object.
func (in *VirtualCluster) SetConditions(conditions Conditions) {
	in.Status.Conditions = conditions
}

type VirtualClusterSpec struct {
	// VirtualClusterCommonSpec defines virtual cluster spec that is common between the virtual
	// cluster templates, and virtual cluster
	VirtualClusterCommonSpec `json:",inline"`

	// DEPRECATED: don't use anymore
	// A label selector to select the virtual cluster pod to route
	// incoming requests to.
	// +optional
	Pod *PodSelector `json:"pod,omitempty"`

	// DEPRECATED: don't use anymore
	// A reference to the cluster admin kube config. This is needed for
	// the cli & ui to access the virtual clusters
	// +optional
	KubeConfigRef *SecretRef `json:"kubeConfigRef,omitempty"`
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

// Chart describes a chart
type Chart struct {
	// Name is the chart name in the repository
	Name string `json:"name,omitempty"`

	// Version is the chart version in the repository
	// +optional
	Version string `json:"version,omitempty"`

	// RepoURL is the repo url where the chart can be found
	// +optional
	RepoURL string `json:"repoURL,omitempty"`

	// The username that is required for this repository
	// +optional
	Username string `json:"username,omitempty"`

	// The username that is required for this repository
	// +optional
	UsernameRef *ChartSecretRef `json:"usernameRef,omitempty"`

	// The password that is required for this repository
	// +optional
	Password string `json:"password,omitempty"`

	// The password that is required for this repository
	// +optional
	PasswordRef *ChartSecretRef `json:"passwordRef,omitempty"`

	// If tls certificate checks for the chart download should be skipped
	// +optional
	InsecureSkipTlsVerify bool `json:"insecureSkipTlsVerify,omitempty"`
}

type ChartSecretRef struct {
	// ProjectSecretRef holds the reference to a project secret
	// +optional
	ProjectSecretRef *ProjectSecretRef `json:"projectSecretRef,omitempty"`
}

type ProjectSecretRef struct {
	// Project is the project name where the secret is located in.
	// +optional
	Project string `json:"project,omitempty"`

	// Name of the project secret to use.
	// +optional
	Name string `json:"name,omitempty"`

	// Key of the project secret to use.
	// +optional
	Key string `json:"key,omitempty"`
}

type TemplateHelmChart struct {
	Chart `json:",inline"`

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

// SecretRef is the reference to a secret containing the user password
type SecretRef struct {
	// +optional
	SecretName string `json:"secretName,omitempty"`
	// +optional
	SecretNamespace string `json:"secretNamespace,omitempty"`
	// +optional
	Key string `json:"key,omitempty"`
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
	Conditions Conditions `json:"conditions,omitempty"`

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

// VirtualClusterList contains a list of User
type VirtualClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualCluster{}, &VirtualClusterList{})
}
