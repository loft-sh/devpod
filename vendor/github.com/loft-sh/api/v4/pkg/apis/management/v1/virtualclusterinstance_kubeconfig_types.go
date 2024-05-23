package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterInstanceKubeConfig holds kube config request and response data for virtual clusters
// +subresource-request
type VirtualClusterInstanceKubeConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterInstanceKubeConfigSpec   `json:"spec,omitempty"`
	Status VirtualClusterInstanceKubeConfigStatus `json:"status,omitempty"`
}

type VirtualClusterInstanceKubeConfigSpec struct {
	// CertificateTTL holds the ttl (in seconds) to set for the certificate associated with the
	// returned kubeconfig.
	// This field is optional, if no value is provided, the certificate TTL will be set to one day.
	// If set to zero, this will cause loft to pass nil to the certificate signing request, which
	// will result in the certificate being valid for the clusters `cluster-signing-duration` value
	// which is typically one year.
	// +optional
	CertificateTTL *int32 `json:"certificateTTL,omitempty"`
}

type VirtualClusterInstanceKubeConfigStatus struct {
	// KubeConfig holds the final kubeconfig output
	KubeConfig string `json:"kubeConfig,omitempty"`
}
