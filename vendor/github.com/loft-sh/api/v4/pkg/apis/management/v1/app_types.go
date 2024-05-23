package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:method=GetCredentials,verb=get,subresource=credentials,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.AppCredentials
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// App holds the information
// +k8s:openapi-gen=true
// +resource:path=apps,rest=AppREST
// +subresource:request=AppCredentials,path=credentials,kind=AppCredentials,rest=AppCredentialsREST
type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppSpec   `json:"spec,omitempty"`
	Status AppStatus `json:"status,omitempty"`
}

// AppSpec holds the specification
type AppSpec struct {
	storagev1.AppSpec `json:",inline"`
}

// AppStatus holds the status
type AppStatus struct {
	storagev1.AppStatus `json:",inline"`
}

func (a *App) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *App) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *App) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *App) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
