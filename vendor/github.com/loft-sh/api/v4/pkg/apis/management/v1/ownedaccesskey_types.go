package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OwnedAccessKey is an access key that is owned by the current user
// +k8s:openapi-gen=true
// +resource:path=ownedaccesskeys,rest=OwnedAccessKeyREST
type OwnedAccessKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OwnedAccessKeySpec   `json:"spec,omitempty"`
	Status OwnedAccessKeyStatus `json:"status,omitempty"`
}

type OwnedAccessKeySpec struct {
	storagev1.AccessKeySpec `json:",inline"`
}

type OwnedAccessKeyStatus struct {
	storagev1.AccessKeyStatus `json:",inline"`
}
