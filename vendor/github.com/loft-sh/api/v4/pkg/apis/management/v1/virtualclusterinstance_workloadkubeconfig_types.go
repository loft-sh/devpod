package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterInstanceWorkloadKubeConfig holds kube config request and response data for virtual clusters
// +subresource-request
type VirtualClusterInstanceWorkloadKubeConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// KubeConfig holds the workload cluster's kubeconfig to access the virtual cluster
	// +optional
	KubeConfig string `json:"kubeConfig,omitempty"`

	// Token holds the service account token vcluster should use to connect to the remote cluster
	// +optional
	Token string `json:"token,omitempty"`
}
