package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectImportVirtualCluster holds project vcluster import information
// +subresource-request
type ProjectImportVirtualCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// SourceVirtualCluster is the virtual cluster to import into this project
	SourceVirtualCluster ProjectImportVirtualClusterSource `json:"sourceVirtualCluster"`

	// UpgradeToPro indicates whether we should upgrade to Pro on import
	// +optional
	UpgradeToPro bool `json:"upgradeToPro,omitempty"`

	// SkipHelmDeploy will skip management of the vClusters helm deployment
	// +optional
	SkipHelmDeploy bool `json:"skipHelmDeploy,omitempty"`
}

type ProjectImportVirtualClusterSource struct {
	// Name of the virtual cluster to import
	Name string `json:"name,omitempty"`

	// Namespace of the virtual cluster to import
	Namespace string `json:"namespace,omitempty"`

	// Cluster name of the cluster the virtual cluster is running on
	Cluster string `json:"cluster,omitempty"`

	// Owner of the virtual cluster to import
	// +optional
	Owner *storagev1.UserOrTeam `json:"owner,omitempty"`

	// ImportName is an optional name to use as the virtualclusterinstance name, if not provided
	// the vcluster name will be used
	// +optional
	ImportName string `json:"importName,omitempty"`
}
