package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DevPodWorkspaceEnvironmentSource
// +k8s:openapi-gen=true
type DevPodEnvironmentTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DevPodEnvironmentTemplateSpec `json:"spec,omitempty"`
}

func (a *DevPodEnvironmentTemplate) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *DevPodEnvironmentTemplate) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *DevPodEnvironmentTemplate) GetAccess() []Access {
	return a.Spec.Access
}

func (a *DevPodEnvironmentTemplate) SetAccess(access []Access) {
	a.Spec.Access = access
}

type DevPodEnvironmentTemplateSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Git holds configuration for git environment spec source
	// +optional
	Git GitEnvironmentTemplate `json:"git,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Access to the DevPod machine instance object itself
	// +optional
	Access []Access `json:"access,omitempty"`

	// Versions are different versions of the template that can be referenced as well
	// +optional
	Versions []DevPodEnvironmentTemplateVersion `json:"versions,omitempty"`
}

// GitEnvironmentTemplate stores configuration of Git environment template source
type GitEnvironmentTemplate struct {
	// Repository stores repository URL for Git environment spec source
	Repository string `json:"repository"`

	// Revision stores revision to checkout in repository
	// +optional
	Revision string `json:"revision,omitempty"`

	// SubPath stores subpath within Repositor where environment spec is
	// +optional
	SubPath string `json:"subpath,omitempty"`

	// UseProjectGitCredentials specifies if the project git credentials should be used instead of local ones for this environment
	// +optional
	UseProjectGitCredentials bool `json:"useProjectGitCredentials,omitempty"`
}

type DevPodEnvironmentTemplateVersion struct {
	// Git holds the GitEnvironmentTemplate
	// +optional
	Git GitEnvironmentTemplate `json:"git,omitempty"`

	// Version is the version. Needs to be in X.X.X format.
	// +optional
	Version string `json:"version,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DevPodEnvironmentTemplateList contains a list of DevPodEnvironmentTemplate objects
type DevPodEnvironmentTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DevPodEnvironmentTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DevPodEnvironmentTemplate{}, &DevPodEnvironmentTemplateList{})
}
