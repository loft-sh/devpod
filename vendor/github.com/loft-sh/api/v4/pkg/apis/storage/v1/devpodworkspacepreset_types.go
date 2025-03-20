package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DevPodWorkspacePreset
// +k8s:openapi-gen=true
type DevPodWorkspacePreset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevPodWorkspacePresetSpec   `json:"spec,omitempty"`
	Status DevPodWorkspacePresetStatus `json:"status,omitempty"`
}

func (a *DevPodWorkspacePreset) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *DevPodWorkspacePreset) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *DevPodWorkspacePreset) GetAccess() []Access {
	return a.Spec.Access
}

func (a *DevPodWorkspacePreset) SetAccess(access []Access) {
	a.Spec.Access = access
}

type DevPodWorkspacePresetSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Source stores inline path of project source
	Source *DevPodWorkspacePresetSource `json:"source"`

	// InfrastructureRef stores reference to DevPodWorkspaceTemplate to use
	InfrastructureRef *TemplateRef `json:"infrastructureRef"`

	// EnvironmentRef stores reference to DevPodEnvironmentTemplate
	// +optional
	EnvironmentRef *EnvironmentRef `json:"environmentRef,omitempty"`

	// UseProjectGitCredentials specifies if the project git credentials should be used instead of local ones for this environment
	// +optional
	UseProjectGitCredentials bool `json:"useProjectGitCredentials,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Access to the DevPod machine instance object itself
	// +optional
	Access []Access `json:"access,omitempty"`

	// Versions are different versions of the template that can be referenced as well
	// +optional
	Versions []DevPodWorkspacePresetVersion `json:"versions,omitempty"`
}

type DevPodWorkspacePresetSource struct {
	// Git stores path to git repo to use as workspace source
	// +optional
	Git string `json:"git,omitempty"`

	// Image stores container image to use as workspace source
	// +optional
	Image string `json:"image,omitempty"`
}

type DevPodWorkspacePresetVersion struct {
	// Version is the version. Needs to be in X.X.X format.
	// +optional
	Version string `json:"version,omitempty"`

	// Source stores inline path of project source
	// +optional
	Source *DevPodWorkspacePresetSource `json:"source,omitempty"`

	// InfrastructureRef stores reference to DevPodWorkspaceTemplate to use
	// +optional
	InfrastructureRef *TemplateRef `json:"infrastructureRef,omitempty"`

	// EnvironmentRef stores reference to DevPodEnvironmentTemplate
	// +optional
	EnvironmentRef *EnvironmentRef `json:"environmentRef,omitempty"`
}

// DevPodWorkspacePresetStatus holds the status
type DevPodWorkspacePresetStatus struct {
}

type WorkspaceRef struct {
	// Name is the name of DevPodWorkspaceTemplate this references
	Name string `json:"name"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// DevPodWorkspacePresetList contains a list of DevPodWorkspacePreset objects
type DevPodWorkspacePresetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DevPodWorkspacePreset `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DevPodWorkspacePreset{}, &DevPodWorkspacePresetList{})
}
