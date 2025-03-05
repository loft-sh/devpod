package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:method=ListMembers,verb=get,subresource=members,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.ProjectMembers
// +genclient:method=ListTemplates,verb=get,subresource=templates,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.ProjectTemplates
// +genclient:method=ListClusters,verb=get,subresource=clusters,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.ProjectClusters
// +genclient:method=MigrateVirtualClusterInstance,verb=create,subresource=migratevirtualclusterinstance,input=github.com/loft-sh/api/v4/pkg/apis/management/v1.ProjectMigrateVirtualClusterInstance,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.ProjectMigrateVirtualClusterInstance
// +genclient:method=ImportSpace,verb=create,subresource=importspace,input=github.com/loft-sh/api/v4/pkg/apis/management/v1.ProjectImportSpace,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.ProjectImportSpace
// +genclient:method=MigrateSpaceInstance,verb=create,subresource=migratespaceinstance,input=github.com/loft-sh/api/v4/pkg/apis/management/v1.ProjectMigrateSpaceInstance,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.ProjectMigrateSpaceInstance
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Project holds the Project information
// +k8s:openapi-gen=true
// +resource:path=projects,rest=ProjectREST,statusRest=ProjectStatusREST
// +subresource:request=ProjectCharts,path=charts,kind=ProjectCharts,rest=ProjectChartsREST
// +subresource:request=ProjectTemplates,path=templates,kind=ProjectTemplates,rest=ProjectTemplatesREST
// +subresource:request=ProjectMembers,path=members,kind=ProjectMembers,rest=ProjectMembersREST
// +subresource:request=ProjectClusters,path=clusters,kind=ProjectClusters,rest=ProjectClustersREST
// +subresource:request=ProjectChartInfo,path=chartinfo,kind=ProjectChartInfo,rest=ProjectChartInfoREST
// +subresource:request=ProjectMigrateVirtualClusterInstance,path=migratevirtualclusterinstance,kind=ProjectMigrateVirtualClusterInstance,rest=ProjectMigrateVirtualClusterInstanceREST
// +subresource:request=ProjectImportSpace,path=importspace,kind=ProjectImportSpace,rest=ProjectImportSpaceREST
// +subresource:request=ProjectMigrateSpaceInstance,path=migratespaceinstance,kind=ProjectMigrateSpaceInstance,rest=ProjectMigrateSpaceInstanceREST
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// ProjectSpec holds the specification
type ProjectSpec struct {
	storagev1.ProjectSpec `json:",inline"`
}

// ProjectStatus holds the status
type ProjectStatus struct {
	storagev1.ProjectStatus `json:",inline"`
}

func (a *Project) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *Project) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *Project) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *Project) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
