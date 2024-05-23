package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResetAccessKey is an access key that is owned by another user
// +k8s:openapi-gen=true
// +resource:path=resetaccesskeys,rest=ResetAccessKeyREST
type ResetAccessKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResetAccessKeySpec   `json:"spec,omitempty"`
	Status ResetAccessKeyStatus `json:"status,omitempty"`
}

type ResetAccessKeySpec struct {
	storagev1.AccessKeySpec `json:",inline"`
}

type ResetAccessKeyStatus struct {
	storagev1.AccessKeyStatus `json:",inline"`
}
