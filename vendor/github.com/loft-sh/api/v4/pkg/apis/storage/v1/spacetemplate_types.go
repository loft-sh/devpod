package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SpaceTemplate holds the space template information
// +k8s:openapi-gen=true
type SpaceTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpaceTemplateSpec   `json:"spec,omitempty"`
	Status SpaceTemplateStatus `json:"status,omitempty"`
}

func (a *SpaceTemplate) GetVersions() []VersionAccessor {
	var retVersions []VersionAccessor
	for _, v := range a.Spec.Versions {
		b := v
		retVersions = append(retVersions, &b)
	}

	return retVersions
}

func (a *SpaceTemplateVersion) GetVersion() string {
	return a.Version
}

func (a *SpaceTemplate) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *SpaceTemplate) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *SpaceTemplate) GetAccess() []Access {
	return a.Spec.Access
}

func (a *SpaceTemplate) SetAccess(access []Access) {
	a.Spec.Access = access
}

// SpaceTemplateSpec holds the specification
type SpaceTemplateSpec struct {
	// DisplayName is the name that is shown in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes the space template
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Template holds the space template
	// +optional
	Template SpaceTemplateDefinition `json:"template,omitempty"`

	// Parameters define additional app parameters that will set helm values
	// +optional
	Parameters []AppParameter `json:"parameters,omitempty"`

	// Versions are different space template versions that can be referenced as well
	// +optional
	Versions []SpaceTemplateVersion `json:"versions,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`
}

type SpaceTemplateVersion struct {
	// Template holds the space template
	// +optional
	Template SpaceTemplateDefinition `json:"template,omitempty"`

	// Parameters define additional app parameters that will set helm values
	// +optional
	Parameters []AppParameter `json:"parameters,omitempty"`

	// Version is the version. Needs to be in X.X.X format.
	// +optional
	Version string `json:"version,omitempty"`
}

type SpaceTemplateDefinition struct {
	// The space metadata
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	TemplateMetadata `json:"metadata,omitempty"`

	// InstanceTemplate holds the space instance template
	// +optional
	InstanceTemplate SpaceInstanceTemplateDefinition `json:"instanceTemplate,omitempty"`

	// Objects are Kubernetes style yamls that should get deployed into the virtual cluster
	// +optional
	Objects string `json:"objects,omitempty"`

	// Charts are helm charts that should get deployed
	// +optional
	Charts []TemplateHelmChart `json:"charts,omitempty"`

	// Apps specifies the apps that should get deployed by this template
	// +optional
	Apps []AppReference `json:"apps,omitempty"`

	// The space access
	// +optional
	Access *InstanceAccess `json:"access,omitempty"`
}

// SpaceInstanceTemplateDefinition holds the space instance template
type SpaceInstanceTemplateDefinition struct {
	// The space instance metadata
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	TemplateMetadata `json:"metadata,omitempty"`
}

// SpaceTemplateStatus holds the status
type SpaceTemplateStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SpaceTemplateList contains a list of SpaceTemplate
type SpaceTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SpaceTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SpaceTemplate{}, &SpaceTemplateList{})
}
