package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterTemplate holds the virtualClusterTemplate information
// +k8s:openapi-gen=true
type VirtualClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterTemplateSpec   `json:"spec,omitempty"`
	Status VirtualClusterTemplateStatus `json:"status,omitempty"`
}

func (a *VirtualClusterTemplate) GetVersions() []VersionAccessor {
	var retVersions []VersionAccessor
	for _, v := range a.Spec.Versions {
		b := v
		retVersions = append(retVersions, &b)
	}

	return retVersions
}

func (a *VirtualClusterTemplateVersion) GetVersion() string {
	return a.Version
}

func (a *VirtualClusterTemplate) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *VirtualClusterTemplate) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *VirtualClusterTemplate) GetAccess() []Access {
	return a.Spec.Access
}

func (a *VirtualClusterTemplate) SetAccess(access []Access) {
	a.Spec.Access = access
}

// VirtualClusterTemplateSpec holds the specification
type VirtualClusterTemplateSpec struct {
	// DisplayName is the name that is shown in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes the virtual cluster template
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Template holds the virtual cluster template
	// +optional
	Template VirtualClusterTemplateDefinition `json:"template,omitempty"`

	// Parameters define additional app parameters that will set helm values
	// +optional
	Parameters []AppParameter `json:"parameters,omitempty"`

	// Versions are different versions of the template that can be referenced as well
	// +optional
	Versions []VirtualClusterTemplateVersion `json:"versions,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`

	// =======================
	// DEPRECATED FIELDS BELOW
	// =======================

	// DEPRECATED: SpaceTemplate to use to create the virtual cluster space if it does not exist
	// +optional
	SpaceTemplateRef *VirtualClusterTemplateSpaceTemplateRef `json:"spaceTemplateRef,omitempty"`
}

type VirtualClusterTemplateVersion struct {
	// Template holds the space template
	// +optional
	Template VirtualClusterTemplateDefinition `json:"template,omitempty"`

	// Parameters define additional app parameters that will set helm values
	// +optional
	Parameters []AppParameter `json:"parameters,omitempty"`

	// Version is the version. Needs to be in X.X.X format.
	// +optional
	Version string `json:"version,omitempty"`
}

type VirtualClusterTemplateSpaceTemplateRef struct {
	// Name of the space template
	// +optional
	Name string `json:"name,omitempty"`
}

type VirtualClusterTemplateDefinition struct {
	// The virtual cluster metadata
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	TemplateMetadata `json:"metadata,omitempty"`

	// InstanceTemplate holds the virtual cluster instance template
	// +optional
	InstanceTemplate VirtualClusterInstanceTemplateDefinition `json:"instanceTemplate,omitempty"`

	// VirtualClusterCommonSpec defines virtual cluster spec that is common between the virtual
	// cluster templates, and virtual cluster
	VirtualClusterCommonSpec `json:",inline"`

	// SpaceTemplate holds the space template
	// +optional
	SpaceTemplate VirtualClusterSpaceTemplateDefinition `json:"spaceTemplate,omitempty"`
}

type VirtualClusterSpaceTemplateDefinition struct {
	// The space metadata
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	TemplateMetadata `json:"metadata,omitempty"`

	// Objects are Kubernetes style yamls that should get deployed into the virtual cluster namespace
	// +optional
	Objects string `json:"objects,omitempty"`

	// Charts are helm charts that should get deployed
	// +optional
	Charts []TemplateHelmChart `json:"charts,omitempty"`

	// Apps specifies the apps that should get deployed by this template
	// +optional
	Apps []AppReference `json:"apps,omitempty"`
}

// VirtualClusterInstanceTemplateDefinition holds the virtual cluster instance template
type VirtualClusterInstanceTemplateDefinition struct {
	// The virtual cluster instance metadata
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	TemplateMetadata `json:"metadata,omitempty"`
}

type TemplateMetadata struct {
	// Labels are labels on the object
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are annotations on the object
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// VirtualClusterTemplateStatus holds the status
type VirtualClusterTemplateStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterTemplateList contains a list of VirtualClusterTemplate
type VirtualClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualClusterTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualClusterTemplate{}, &VirtualClusterTemplateList{})
}
