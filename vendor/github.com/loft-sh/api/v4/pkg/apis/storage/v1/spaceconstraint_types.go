package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SpaceConstraint holds the global space constraint information
// +k8s:openapi-gen=true
type SpaceConstraint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpaceConstraintSpec   `json:"spec,omitempty"`
	Status SpaceConstraintStatus `json:"status,omitempty"`
}

func (a *SpaceConstraint) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *SpaceConstraint) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *SpaceConstraint) GetAccess() []Access {
	return a.Spec.Access
}

func (a *SpaceConstraint) SetAccess(access []Access) {
	a.Spec.Access = access
}

type SpaceConstraintSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes a space constraint object
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Clusters are the clusters this template should be applied on.
	// +optional
	Clusters []string `json:"clusters,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`

	// LocalSpaceConstraintTemplate holds the space constraint template
	// +omitempty
	LocalSpaceConstraintTemplate LocalSpaceConstraintTemplate `json:"localSpaceConstraintTemplate,omitempty"`
}

type LocalSpaceConstraintTemplate struct {
	// Metadata is the metadata of the space constraint object
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Metadata metav1.ObjectMeta `json:"metadata,omitempty"`

	// LocalSpaceConstraintSpec holds the spec of the space constraint in the cluster
	// +optional
	LocalSpaceConstraintSpec LocalSpaceConstraintSpec `json:"spec,omitempty"`
}

type LocalSpaceConstraintSpec struct {
	// DisplayName is the name that should be shown in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description is the description of this object in
	// human-readable text.
	// +optional
	Description string `json:"description,omitempty"`

	// SpaceTemplate holds the space configuration
	// +optional
	SpaceTemplate ConstraintSpaceTemplate `json:"spaceTemplate,omitempty"`

	// Sync specifies if spaces that were created through this space constraint
	// object should get synced with this object.
	// +optional
	Sync bool `json:"sync,omitempty"`
}

// ConstraintSpaceTemplate defines properties how many spaces can be owned by the account and how they should be created
type ConstraintSpaceTemplate struct {
	// The enforced metadata of the space to create. Currently, only annotations and labels are supported
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// This defines the cluster role that will be used for the rolebinding when
	// creating a new space for the selected subjects
	// +optional
	ClusterRole *string `json:"clusterRole,omitempty"`

	// Objects are Kubernetes style yamls that should get deployed into the space
	// +optional
	Objects string `json:"objects,omitempty"`
}

// SpaceConstraintStatus holds the status of a user access
type SpaceConstraintStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SpaceConstraintList contains a list of SpaceConstraint objects
type SpaceConstraintList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SpaceConstraint `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SpaceConstraint{}, &SpaceConstraintList{})
}
