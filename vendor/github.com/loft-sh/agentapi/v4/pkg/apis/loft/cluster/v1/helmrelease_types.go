package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// +resource:path=helmreleases,rest=HelmReleaseREST
type HelmRelease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HelmReleaseSpec   `json:"spec,omitempty"`
	Status HelmReleaseStatus `json:"status,omitempty"`
}

type HelmReleaseSpec struct {
	HelmReleaseConfig `json:",inline"`
}

type HelmReleaseStatus struct {
	// Revision is an int which represents the revision of the release.
	Revision int `json:"version,omitempty"`

	// Info provides information about a release
	// +optional
	Info *Info `json:"info,omitempty"`

	// Metadata provides information about a chart
	// +optional
	Metadata *Metadata `json:"metadata,omitempty"`
}

type HelmReleaseApp struct {
	// Name is the name of the app this release refers to
	// +optional
	Name string `json:"name,omitempty"`

	// Revision is the revision of the app this release refers to
	// +optional
	Revision string `json:"version,omitempty"`
}

type HelmReleaseConfig struct {
	// Chart holds information about a chart that should get deployed
	// +optional
	Chart Chart `json:"chart,omitempty"`

	// Manifests holds kube manifests that will be deployed as a chart
	// +optional
	Manifests string `json:"manifests,omitempty"`

	// Bash holds the bash script to execute in a container in the target
	// +optional
	Bash *Bash `json:"bash,omitempty"`

	// Values is the set of extra Values added to the chart.
	// These values merge with the default values inside of the chart.
	// You can use golang templating in here with values from parameters.
	// +optional
	Values string `json:"values,omitempty"`

	// Parameters are additional helm chart values that will get merged
	// with config and are then used to deploy the helm chart.
	// +optional
	Parameters string `json:"parameters,omitempty"`

	// Annotations are extra annotations for this helm release
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
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

type Bash struct {
	// Script is the script to execute.
	// +optional
	Script string `json:"script,omitempty"`

	// Image is the image to use for this app
	// +optional
	Image string `json:"image,omitempty"`

	// ClusterRole is the cluster role to use for this job
	// +optional
	ClusterRole string `json:"clusterRole,omitempty"`
}

// Info describes release information.
type Info struct {
	// FirstDeployed is when the release was first deployed.
	// +optional
	FirstDeployed metav1.Time `json:"first_deployed,omitempty"`
	// LastDeployed is when the release was last deployed.
	// +optional
	LastDeployed metav1.Time `json:"last_deployed,omitempty"`
	// Deleted tracks when this object was deleted.
	// +optional
	Deleted metav1.Time `json:"deleted"`
	// Description is human-friendly "log entry" about this release.
	// +optional
	Description string `json:"description,omitempty"`
	// Status is the current state of the release
	// +optional
	Status Status `json:"status,omitempty"`
	// Contains the rendered templates/NOTES.txt if available
	// +optional
	Notes string `json:"notes,omitempty"`
}

// Status is the status of a release
type Status string

// Describe the status of a release
// NOTE: Make sure to update cmd/helm/status.go when adding or modifying any of these statuses.
const (
	// StatusUnknown indicates that a release is in an uncertain state.
	StatusUnknown Status = "unknown"
	// StatusDeployed indicates that the release has been pushed to Kubernetes.
	StatusDeployed Status = "deployed"
	// StatusUninstalled indicates that a release has been uninstalled from Kubernetes.
	StatusUninstalled Status = "uninstalled"
	// StatusSuperseded indicates that this release object is outdated and a newer one exists.
	StatusSuperseded Status = "superseded"
	// StatusFailed indicates that the release was not successfully deployed.
	StatusFailed Status = "failed"
	// StatusUninstalling indicates that a uninstall operation is underway.
	StatusUninstalling Status = "uninstalling"
	// StatusPendingInstall indicates that an install operation is underway.
	StatusPendingInstall Status = "pending-install"
	// StatusPendingUpgrade indicates that an upgrade operation is underway.
	StatusPendingUpgrade Status = "pending-upgrade"
	// StatusPendingRollback indicates that an rollback operation is underway.
	StatusPendingRollback Status = "pending-rollback"
)

func (x Status) String() string { return string(x) }

// Maintainer describes a Chart maintainer.
type Maintainer struct {
	// Name is a user name or organization name
	// +optional
	Name string `json:"name,omitempty"`
	// Email is an optional email address to contact the named maintainer
	// +optional
	Email string `json:"email,omitempty"`
	// URL is an optional URL to an address for the named maintainer
	// +optional
	URL string `json:"url,omitempty"`
}

// Metadata for a Chart file. This models the structure of a Chart.yaml file.
type Metadata struct {
	// The name of the chart
	// +optional
	Name string `json:"name,omitempty"`
	// The URL to a relevant project page, git repo, or contact person
	// +optional
	Home string `json:"home,omitempty"`
	// Source is the URL to the source code of this chart
	// +optional
	Sources []string `json:"sources,omitempty"`
	// A SemVer 2 conformant version string of the chart
	// +optional
	Version string `json:"version,omitempty"`
	// A one-sentence description of the chart
	// +optional
	Description string `json:"description,omitempty"`
	// A list of string keywords
	// +optional
	Keywords []string `json:"keywords,omitempty"`
	// A list of name and URL/email address combinations for the maintainer(s)
	// +optional
	Maintainers []*Maintainer `json:"maintainers,omitempty"`
	// The URL to an icon file.
	// +optional
	Icon string `json:"icon,omitempty"`
	// The API Version of this chart.
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
	// The condition to check to enable chart
	// +optional
	Condition string `json:"condition,omitempty"`
	// The tags to check to enable chart
	// +optional
	Tags string `json:"tags,omitempty"`
	// The version of the application enclosed inside of this chart.
	// +optional
	AppVersion string `json:"appVersion,omitempty"`
	// Whether or not this chart is deprecated
	// +optional
	Deprecated bool `json:"deprecated,omitempty"`
	// Annotations are additional mappings uninterpreted by Helm,
	// made available for inspection by other applications.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// KubeVersion is a SemVer constraint specifying the version of Kubernetes required.
	// +optional
	KubeVersion string `json:"kubeVersion,omitempty"`
	// Specifies the chart type: application or library
	// +optional
	Type string `json:"type,omitempty"`
	// Urls where to find the chart contents
	// +optional
	Urls []string `json:"urls,omitempty"`
}
