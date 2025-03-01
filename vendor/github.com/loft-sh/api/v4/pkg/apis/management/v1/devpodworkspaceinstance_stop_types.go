package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type DevPodWorkspaceInstanceStop struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevPodWorkspaceInstanceStopSpec   `json:"spec,omitempty"`
	Status DevPodWorkspaceInstanceStopStatus `json:"status,omitempty"`
}

type DevPodWorkspaceInstanceStopSpec struct {
	// Options are the options to pass.
	// +optional
	Options string `json:"options,omitempty"`
}

type DevPodWorkspaceInstanceStopStatus struct {
	// TaskID is the id of the task that is running
	// +optional
	TaskID string `json:"taskId,omitempty"`
}
