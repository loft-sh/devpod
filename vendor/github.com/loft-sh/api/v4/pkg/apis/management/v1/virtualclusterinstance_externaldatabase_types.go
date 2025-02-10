package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterExternalDatabase holds kube config request and response data for virtual clusters
// +subresource-request
type VirtualClusterExternalDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterExternalDatabaseSpec   `json:"spec,omitempty"`
	Status VirtualClusterExternalDatabaseStatus `json:"status,omitempty"`
}

type VirtualClusterExternalDatabaseSpec struct {
	// Connector specifies the secret that should be used to connect to an external database server. The connection is
	// used to manage a user and database for the vCluster. A data source endpoint constructed from the created user and
	// database is returned on status. The secret specified by connector should contain the following fields:
	// endpoint - the endpoint where the database server can be accessed
	// user - the database username
	// password - the password for the database username
	// port - the port to be used in conjunction with the endpoint to connect to the databse server. This is commonly
	// 3306
	// +optional
	Connector string `json:"connector,omitempty"`
}

type VirtualClusterExternalDatabaseStatus struct {
	// DataSource holds a datasource endpoint constructed from the vCluster's designated user and database. The user and
	// database are created from the given connector.
	DataSource string `json:"dataSource,omitempty"`
}
