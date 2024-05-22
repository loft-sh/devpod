package v1

import (
	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// App holds the app information
// +k8s:openapi-gen=true
type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppSpec   `json:"spec,omitempty"`
	Status AppStatus `json:"status,omitempty"`
}

func (a *App) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *App) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *App) GetAccess() []Access {
	return a.Spec.Access
}

func (a *App) SetAccess(access []Access) {
	a.Spec.Access = access
}

// AppSpec holds the specification
type AppSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes an app
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Clusters are the clusters this app can be installed in.
	// +optional
	Clusters []string `json:"clusters,omitempty"`

	// RecommendedApp specifies where this app should show up as recommended app
	// +optional
	RecommendedApp []RecommendedApp `json:"recommendedApp,omitempty"`

	// AppConfig is the app configuration
	AppConfig `json:",inline"`

	// Versions are different app versions that can be referenced
	// +optional
	Versions []AppVersion `json:"versions,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`

	// =======================
	// DEPRECATED FIELDS BELOW
	// =======================

	// DEPRECATED: Use config instead
	// manifest represents kubernetes resources that will be deployed into the target namespace
	// +optional
	Manifests string `json:"manifests,omitempty"`

	// DEPRECATED: Use config instead
	// helm defines the configuration for a helm deployment
	// +optional
	Helm *HelmConfiguration `json:"helm,omitempty"`
}

type AppVersion struct {
	// AppConfig is the app configuration
	AppConfig `json:",inline"`

	// Version is the version. Needs to be in X.X.X format.
	// +optional
	Version string `json:"version,omitempty"`
}

type AppConfig struct {
	// DefaultNamespace is the default namespace this app should installed
	// in.
	// +optional
	DefaultNamespace string `json:"defaultNamespace,omitempty"`

	// Readme is a longer markdown string that describes the app.
	// +optional
	Readme string `json:"readme,omitempty"`

	// Icon holds an URL to the app icon
	// +optional
	Icon string `json:"icon,omitempty"`

	// Config is the helm config to use to deploy the helm release
	// +optional
	Config clusterv1.HelmReleaseConfig `json:"config,omitempty"`

	// Wait determines if Loft should wait during deploy for the app to become ready
	// +optional
	Wait bool `json:"wait,omitempty"`

	// Timeout is the time to wait for any individual Kubernetes operation (like Jobs for hooks) (default 5m0s)
	// +optional
	Timeout string `json:"timeout,omitempty"`

	// Parameters define additional app parameters that will set helm values
	// +optional
	Parameters []AppParameter `json:"parameters,omitempty"`

	// =======================
	// DEPRECATED FIELDS BELOW
	// =======================

	// DEPRECATED: Use config.bash instead
	// StreamContainer can be used to stream a containers logs instead of the helm output.
	// +optional
	// +internal
	StreamContainer *StreamContainer `json:"streamContainer,omitempty"`
}

type AppParameter struct {
	// Variable is the path of the variable. Can be foo or foo.bar for nested objects.
	// +optional
	Variable string `json:"variable,omitempty"`

	// Label is the label to show for this parameter
	// +optional
	Label string `json:"label,omitempty"`

	// Description is the description to show for this parameter
	// +optional
	Description string `json:"description,omitempty"`

	// Type of the parameter. Can be one of:
	// string, multiline, boolean, number and password
	// +optional
	Type string `json:"type,omitempty"`

	// Options is a slice of strings, where each string represents a mutually exclusive choice.
	// +optional
	Options []string `json:"options,omitempty"`

	// Min is the minimum number if type is number
	// +optional
	Min *int `json:"min,omitempty"`

	// Max is the maximum number if type is number
	// +optional
	Max *int `json:"max,omitempty"`

	// Required specifies if this parameter is required
	// +optional
	Required bool `json:"required,omitempty"`

	// DefaultValue is the default value if none is specified
	// +optional
	DefaultValue string `json:"defaultValue,omitempty"`

	// Placeholder shown in the UI
	// +optional
	Placeholder string `json:"placeholder,omitempty"`

	// Invalidation regex that if matched will reject the input
	// +optional
	Invalidation string `json:"invalidation,omitempty"`

	// Validation regex that if matched will allow the input
	// +optional
	Validation string `json:"validation,omitempty"`

	// Section where this app should be displayed. Apps with the same section name will be grouped together
	// +optional
	Section string `json:"section,omitempty"`
}

type UserOrTeam struct {
	// User specifies a Loft user.
	// +optional
	User string `json:"user,omitempty"`

	// Team specifies a Loft team.
	// +optional
	Team string `json:"team,omitempty"`
}

// HelmConfiguration holds the helm configuration
type HelmConfiguration struct {
	// Name of the chart to deploy
	Name string `json:"name"`

	// The additional helm values to use. Expected block string
	// +optional
	Values string `json:"values,omitempty"`

	// Version is the version of the chart to deploy
	// +optional
	Version string `json:"version,omitempty"`

	// The repo url to use
	// +optional
	RepoURL string `json:"repoUrl,omitempty"`

	// The username to use for the selected repository
	// +optional
	Username string `json:"username,omitempty"`

	// The password to use for the selected repository
	// +optional
	Password string `json:"password,omitempty"`

	// Determines if the remote location uses an insecure
	// TLS certificate.
	// +optional
	Insecure bool `json:"insecure,omitempty"`
}

// AppStatus holds the status
type AppStatus struct {
}

// RecommendedApp describes where an app can be displayed as recommended app
type RecommendedApp string

// Describe the status of a release
// NOTE: Make sure to update cmd/helm/status.go when adding or modifying any of these statuses.
const (
	// RecommendedAppCluster indicates that an app should be displayed as recommended app in the cluster view
	RecommendedAppCluster RecommendedApp = "cluster"
	// RecommendedAppSpace indicates that an app should be displayed as recommended app in the space view
	RecommendedAppSpace RecommendedApp = "space"
	// RecommendedAppVirtualCluster indicates that an app should be displayed as recommended app in the virtual cluster view
	RecommendedAppVirtualCluster RecommendedApp = "virtualcluster"
)

func (x RecommendedApp) String() string { return string(x) }

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppList contains a list of App
type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []App `json:"items"`
}

func init() {
	SchemeBuilder.Register(&App{}, &AppList{})
}
