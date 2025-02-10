package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterQuota is the Schema for the cluster quotas api
// +k8s:openapi-gen=true
type ClusterQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterQuotaSpec `json:"spec,omitempty"`

	// +optional
	Status ClusterQuotaStatus `json:"status,omitempty"`
}

// ClusterQuotasStatusByNamespace bundles multiple resource quota status
type ClusterQuotasStatusByNamespace []ClusterQuotaStatusByNamespace

// ClusterQuotaSpec defines the desired state of ClusterQuota
type ClusterQuotaSpec struct {
	// User is the name of the user this quota should apply to
	// +optional
	User string `json:"user,omitempty"`

	// Team is the name of the team this quota should apply to
	// +optional
	Team string `json:"team,omitempty"`

	// Project is the project that this cluster quota should apply to
	// +optional
	Project string `json:"project,omitempty"`

	// quota is the quota definition with all the limits and selectors
	// +optional
	Quota corev1.ResourceQuotaSpec `json:"quota,omitempty"`
}

// ClusterQuotaStatus defines the observed state of ClusterQuota
type ClusterQuotaStatus struct {
	// Total defines the actual enforced quota and its current usage across all projects
	// +optional
	Total corev1.ResourceQuotaStatus `json:"total"`

	// Namespaces slices the usage by project.  This division allows for quick resolution of
	// deletion reconciliation inside of a single project without requiring a recalculation
	// across all projects.  This can be used to pull the deltas for a given project.
	// +optional
	// +nullable
	Namespaces ClusterQuotasStatusByNamespace `json:"namespaces"`
}

// ClusterQuotaStatusByNamespace holds the status of a specific namespace
type ClusterQuotaStatusByNamespace struct {
	// Namespace of the account this account quota applies to
	Namespace string `json:"namespace"`

	// Status indicates how many resources have been consumed by this project
	// +optional
	Status corev1.ResourceQuotaStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterQuotaList contains a list of ClusterQuota
type ClusterQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterQuota `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterQuota{}, &ClusterQuotaList{})
}
