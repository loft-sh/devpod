package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterAccess holds the global cluster access information
// +k8s:openapi-gen=true
type ClusterAccess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterAccessSpec   `json:"spec,omitempty"`
	Status ClusterAccessStatus `json:"status,omitempty"`
}

func (a *ClusterAccess) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *ClusterAccess) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *ClusterAccess) GetAccess() []Access {
	return a.Spec.Access
}

func (a *ClusterAccess) SetAccess(access []Access) {
	a.Spec.Access = access
}

type ClusterAccessSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes a cluster access object
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

	// LocalClusterAccessTemplate holds the cluster access template
	// +omitempty
	LocalClusterAccessTemplate LocalClusterAccessTemplate `json:"localClusterAccessTemplate,omitempty"`
}

type LocalClusterAccessTemplate struct {
	// Metadata is the metadata of the cluster access object
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Metadata metav1.ObjectMeta `json:"metadata,omitempty"`

	// LocalClusterAccessSpec holds the spec of the cluster access in the cluster
	// +optional
	LocalClusterAccessSpec agentstoragev1.LocalClusterAccessSpec `json:"spec,omitempty"`
}

// ClusterAccessStatus holds the status of a user access
type ClusterAccessStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterAccessList contains a list of ClusterAccess objects
type ClusterAccessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterAccess `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterAccess{}, &ClusterAccessList{})
}
