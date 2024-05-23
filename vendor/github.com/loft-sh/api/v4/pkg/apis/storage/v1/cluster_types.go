package v1

import (
	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster holds the cluster information
// +k8s:openapi-gen=true
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

func (a *Cluster) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *Cluster) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *Cluster) GetAccess() []Access {
	return a.Spec.Access
}

func (a *Cluster) SetAccess(access []Access) {
	a.Spec.Access = access
}

// ClusterSpec holds the cluster specification
type ClusterSpec struct {
	// If specified this name is displayed in the UI instead of the metadata name
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes a cluster access object
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Holds a reference to a secret that holds the kube config to access this cluster
	// +optional
	Config SecretRef `json:"config,omitempty"`

	// Local specifies if it is the local cluster that should be connected, when this is specified, config is optional
	// +optional
	Local bool `json:"local,omitempty"`

	// NetworkPeer specifies if the cluster is connected via tailscale, when this is specified, config is optional
	// +optional
	NetworkPeer bool `json:"networkPeer,omitempty"`

	// The namespace where the cluster components will be installed in
	// +optional
	ManagementNamespace string `json:"managementNamespace,omitempty"`

	// If unusable is true, no spaces or virtual clusters can be scheduled on this cluster.
	// +optional
	Unusable bool `json:"unusable,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`
}

type AllowedClusterAccountTemplate struct {
	// Name is the name of a cluster account template
	// +optional
	Name string `json:"name,omitempty"`
}

// ClusterStatus holds the user status
type ClusterStatus struct {
	// +optional
	Phase ClusterStatusPhase `json:"phase,omitempty"`

	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`
}

// ClusterStatusPhase describes the phase of a cluster
type ClusterStatusPhase string

// These are the valid admin account types
const (
	ClusterStatusPhaseInitializing ClusterStatusPhase = ""
	ClusterStatusPhaseInitialized  ClusterStatusPhase = "Initialized"
	ClusterStatusPhaseFailed       ClusterStatusPhase = "Failed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}

type HelmChart struct {
	// Metadata provides information about a chart
	// +optional
	Metadata clusterv1.Metadata `json:"metadata,omitempty"`

	// Versions holds all chart versions
	// +optional
	Versions []string `json:"versions,omitempty"`

	// Repository is the repository name of this chart
	// +optional
	Repository HelmChartRepository `json:"repository,omitempty"`
}

type HelmChartRepository struct {
	// Name is the name of the repository
	// +optional
	Name string `json:"name,omitempty"`

	// URL is the repository url
	// +optional
	URL string `json:"url,omitempty"`

	// Username of the repository
	// +optional
	Username string `json:"username,omitempty"`

	// Password of the repository
	// +optional
	Password string `json:"password,omitempty"`

	// Insecure specifies if the chart should be retrieved without TLS
	// verification
	// +optional
	Insecure bool `json:"insecure,omitempty"`
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

	// The password that is required for this repository
	// +optional
	Password string `json:"password,omitempty"`
}
