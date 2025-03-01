package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkspaceAccessKey is an access key that is owned by the current user
// +k8s:openapi-gen=true
// +resource:path=workspaceaccesskeys,rest=WorkspaceAccessKeyREST
type WorkspaceAccessKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkspaceAccessKeySpec   `json:"spec,omitempty"`
	Status WorkspaceAccessKeyStatus `json:"status,omitempty"`
}

type WorkspaceAccessKeySpec struct {
	storagev1.AccessKeySpec `json:",inline"`
}

type WorkspaceAccessKeyStatus struct {
	storagev1.AccessKeyStatus `json:",inline"`
}
