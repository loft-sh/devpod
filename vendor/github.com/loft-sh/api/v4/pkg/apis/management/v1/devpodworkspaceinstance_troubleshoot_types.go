package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type DevPodWorkspaceInstanceTroubleshoot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// State holds the workspaces state as given by 'devpod export'
	// +optional
	State string `json:"state,omitempty"`

	// Workspace holds the workspace's instance object data
	// +optional
	Workspace *DevPodWorkspaceInstance `json:"workspace,omitempty"`

	// Template holds the workspace instance's template used to create it.
	// This is the raw template, not the rendered one.
	// +optional
	Template *storagev1.DevPodWorkspaceTemplate `json:"template,omitempty"`

	// Pods is a list of pod objects that are linked to the workspace.
	// +optional
	Pods []corev1.Pod `json:"pods,omitempty"`

	// PVCs is a list of PVC objects that are linked to the workspace.
	// +optional
	PVCs []corev1.PersistentVolumeClaim `json:"pvcs,omitempty"`

	// Netmaps is a list of netmaps that are linked to the workspace.
	// +optional
	Netmaps []string `json:"netmaps,omitempty"`

	// Errors is a list of errors that occurred while trying to collect
	// informations for troubleshooting.
	// +optional
	Errors []string `json:"errors,omitempty"`
}
