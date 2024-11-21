package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DevPodWorkspacePreset
// +k8s:openapi-gen=true
// +resource:path=devpodworkspacepresets,rest=DevPodWorkspacePresetREST
type DevPodWorkspacePreset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevPodWorkspacePresetSpec   `json:"spec,omitempty"`
	Status DevPodWorkspacePresetStatus `json:"status,omitempty"`
}

// DevPodWorkspacePresetSpec holds the specification
type DevPodWorkspacePresetSpec struct {
	storagev1.DevPodWorkspacePresetSpec `json:",inline"`
}

// DevPodWorkspacePresetSource
// +k8s:openapi-gen=true
type DevPodWorkspacePresetSource struct {
	storagev1.DevPodWorkspacePresetSource `json:",inline"`
}

func (a *DevPodWorkspacePreset) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *DevPodWorkspacePreset) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *DevPodWorkspacePreset) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *DevPodWorkspacePreset) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}

// DevPodWorkspacePresetStatus holds the status
type DevPodWorkspacePresetStatus struct{}
