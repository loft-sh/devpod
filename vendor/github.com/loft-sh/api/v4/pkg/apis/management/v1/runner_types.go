package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:method=GetConfig,verb=get,subresource=config,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.RunnerConfig
// +genclient:method=GetAccessKey,verb=get,subresource=accesskey,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.RunnerAccessKey
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Runner holds the Runner information
// +k8s:openapi-gen=true
// +resource:path=runners,rest=RunnerREST,statusRest=RunnerStatusREST
// +subresource:request=RunnerConfig,path=config,kind=RunnerConfig,rest=RunnerConfigREST
// +subresource:request=RunnerAccessKey,path=accesskey,kind=RunnerAccessKey,rest=RunnerAccessKeyREST
type Runner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunnerSpec   `json:"spec,omitempty"`
	Status RunnerStatus `json:"status,omitempty"`
}

// RunnerSpec holds the specification
type RunnerSpec struct {
	storagev1.RunnerSpec `json:",inline"`
}

// RunnerStatus holds the status
type RunnerStatus struct {
	storagev1.RunnerStatus `json:",inline"`
}

func (a *Runner) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *Runner) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *Runner) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *Runner) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *Runner) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *Runner) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
