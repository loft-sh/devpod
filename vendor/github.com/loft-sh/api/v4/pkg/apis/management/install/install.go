package install

import (
	"github.com/loft-sh/api/v4/pkg/apis/management"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/apiserver/pkg/builders"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

func init() {
	InstallOptions(builders.Scheme)
	InstallOptions(builders.ParameterScheme)
	utilruntime.Must(managementv1.RegisterConversions(builders.ParameterScheme))
}

func InstallOptions(scheme *runtime.Scheme) {
	utilruntime.Must(managementv1.InstallOptions(scheme))
	utilruntime.Must(addKnownOptionsTypes(scheme))
}

func addKnownOptionsTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		management.SchemeGroupVersion,
		&management.TaskLogOptions{},
		&management.VirtualClusterInstanceLogOptions{},
		&management.UserSpacesOptions{},
		&management.UserVirtualClustersOptions{},
		&management.UserQuotasOptions{},
		&management.DevPodWorkspaceInstanceLogOptions{},
		&management.DevPodWorkspaceInstanceTasksOptions{},
		&management.DevPodWorkspaceInstanceDownloadOptions{},
	)
	return nil
}
