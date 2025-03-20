package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type DevPodWorkspaceInstanceTasks struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Tasks []DevPodWorkspaceInstanceTask `json:"tasks,omitempty"`
}

type DevPodWorkspaceInstanceTask struct {
	// ID is the id of the task
	ID string `json:"id,omitempty"`

	// Type is the type of the task
	Type string `json:"type,omitempty"`

	// Status is the status of the task
	Status string `json:"status,omitempty"`

	// Result is the result of the task
	Result []byte `json:"result,omitempty"`

	// Logs is the compressed logs of the task
	Logs []byte `json:"logs,omitempty"`

	// CreatedAt is the timestamp when the task was created
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}
