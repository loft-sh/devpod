package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:method=ListAccess,verb=get,subresource=memberaccess,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.ClusterMemberAccess
// +genclient:method=ListMembers,verb=get,subresource=members,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.ClusterMembers
// +genclient:method=ListVirtualClusterDefaults,verb=get,subresource=virtualclusterdefaults,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.ClusterVirtualClusterDefaults
// +genclient:method=GetAgentConfig,verb=get,subresource=agentconfig,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.ClusterAgentConfig
// +genclient:method=GetAccessKey,verb=get,subresource=accesskey,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.ClusterAccessKey
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster holds the cluster information
// +k8s:openapi-gen=true
// +resource:path=clusters,rest=ClusterREST
// +subresource:request=ClusterMemberAccess,path=memberaccess,kind=ClusterMemberAccess,rest=ClusterMemberAccessREST
// +subresource:request=ClusterReset,path=reset,kind=ClusterReset,rest=ClusterResetREST
// +subresource:request=ClusterDomain,path=domain,kind=ClusterDomain,rest=ClusterDomainREST
// +subresource:request=ClusterMembers,path=members,kind=ClusterMembers,rest=ClusterMembersREST
// +subresource:request=ClusterCharts,path=charts,kind=ClusterCharts,rest=ClusterChartsREST
// +subresource:request=ClusterVirtualClusterDefaults,path=virtualclusterdefaults,kind=ClusterVirtualClusterDefaults,rest=ClusterVirtualClusterDefaultsREST
// +subresource:request=ClusterAgentConfig,path=agentconfig,kind=ClusterAgentConfig,rest=ClusterAgentConfigREST
// +subresource:request=ClusterAccessKey,path=accesskey,kind=ClusterAccessKey,rest=ClusterAccessKeyREST
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// ClusterSpec holds the specification
type ClusterSpec struct {
	storagev1.ClusterSpec `json:",inline"`
}

// ClusterStatus holds the status
type ClusterStatus struct {
	storagev1.ClusterStatus `json:",inline"`

	// Online is whether the cluster is currently connected to the coordination
	// server.
	// +optional
	Online bool `json:"online,omitempty"`
}

func (a *Cluster) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *Cluster) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *Cluster) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *Cluster) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
