package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualCluster holds the virtual cluster information
// +k8s:openapi-gen=true
// +resource:path=virtualclusters,rest=VirtualClusterREST
type VirtualCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterSpec   `json:"spec,omitempty"`
	Status VirtualClusterStatus `json:"status,omitempty"`
}

type VirtualClusterSpec struct {
	agentstoragev1.VirtualClusterSpec `json:",inline"`
}

type VirtualClusterStatus struct {
	agentstoragev1.VirtualClusterStatus `json:",inline"`

	// SyncerPod is the syncer pod
	// +optional
	SyncerPod *corev1.Pod `json:"syncerPod,omitempty"`

	// ClusterPod is the cluster pod
	// +optional
	ClusterPod *corev1.Pod `json:"clusterPod,omitempty"`

	// SleepModeConfig is the sleep mode config of the space
	// +optional
	SleepModeConfig *SleepModeConfig `json:"sleepModeConfig,omitempty"`

	// TemplateSyncStatus describes the template sync status
	// +optional
	TemplateSyncStatus *TemplateSyncStatus `json:"templateSyncStatus,omitempty"`
}
