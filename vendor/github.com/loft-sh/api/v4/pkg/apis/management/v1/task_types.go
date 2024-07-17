package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Task holds the Task information
// +k8s:openapi-gen=true
// +resource:path=tasks,rest=TaskREST
// +subresource:request=TaskLog,path=log,kind=TaskLog,rest=TaskLogREST
type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskSpec   `json:"spec,omitempty"`
	Status TaskStatus `json:"status,omitempty"`
}

// TaskSpec holds the specification
type TaskSpec struct {
	storagev1.TaskSpec `json:",inline"`
}

// TaskStatus holds the status
type TaskStatus struct {
	storagev1.TaskStatus `json:",inline"`

	// +optional
	Owner *storagev1.UserOrTeamEntity `json:"owner,omitempty"`

	// +optional
	Cluster *storagev1.EntityInfo `json:"cluster,omitempty"`
}

func (a *Task) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *Task) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *Task) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *Task) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
