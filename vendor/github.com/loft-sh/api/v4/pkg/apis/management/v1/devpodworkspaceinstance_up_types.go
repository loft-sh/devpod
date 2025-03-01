package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type DevPodWorkspaceInstanceUp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevPodWorkspaceInstanceUpSpec   `json:"spec,omitempty"`
	Status DevPodWorkspaceInstanceUpStatus `json:"status,omitempty"`
}

type DevPodWorkspaceInstanceUpSpec struct {
	// Debug includes debug logs.
	// +optional
	Debug bool `json:"debug,omitempty"`

	// Options are the options to pass.
	// +optional
	Options string `json:"options,omitempty"`
}

type DevPodWorkspaceInstanceUpStatus struct {
	// TaskID is the id of the task that is running
	// +optional
	TaskID string `json:"taskId,omitempty"`
}
