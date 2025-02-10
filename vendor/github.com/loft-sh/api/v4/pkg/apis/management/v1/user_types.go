package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:method=GetProfile,verb=get,subresource=profile,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.UserProfile
// +genclient:method=ListClusters,verb=get,subresource=clusters,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.UserClusters
// +genclient:method=ListAccessKeys,verb=get,subresource=accesskeys,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.UserAccessKeys
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// User holds the user information
// +k8s:openapi-gen=true
// +resource:path=users,rest=UserREST
// +subresource:request=UserClusters,path=clusters,kind=UserClusters,rest=UserClustersREST
// +subresource:request=UserProfile,path=profile,kind=UserProfile,rest=UserProfileREST
// +subresource:request=UserAccessKeys,path=accesskeys,kind=UserAccessKeys,rest=UserAccessKeysREST
// +subresource:request=UserPermissions,path=permissions,kind=UserPermissions,rest=UserPermissionsREST
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

type UserSpec struct {
	storagev1.UserSpec `json:",inline"`
}

// UserStatus holds the status of an user
type UserStatus struct {
	storagev1.UserStatus `json:",inline"`
}

func (a *User) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *User) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *User) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *User) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
