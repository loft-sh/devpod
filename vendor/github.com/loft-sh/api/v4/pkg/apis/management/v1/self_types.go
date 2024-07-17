package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Self holds information about the currently logged in user
// +k8s:openapi-gen=true
// +resource:path=selves,rest=SelfREST
type Self struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SelfSpec   `json:"spec,omitempty"`
	Status SelfStatus `json:"status,omitempty"`
}

type SelfSpec struct {
	// AccessKey is an optional access key to use instead of the provided one
	// +optional
	AccessKey string `json:"accessKey,omitempty"`
}

type SelfStatus struct {
	// The name of the currently logged in user
	// +optional
	User *UserInfo `json:"user,omitempty"`

	// The name of the currently logged in team
	// +optional
	Team *storagev1.EntityInfo `json:"team,omitempty"`

	// The name of the currently used access key
	// +optional
	AccessKey string `json:"accessKey,omitempty"`

	// The scope of the currently used access key
	// +optional
	AccessKeyScope *storagev1.AccessKeyScope `json:"accessKeyScope,omitempty"`

	// The type of the currently used access key
	// +optional
	AccessKeyType storagev1.AccessKeyType `json:"accessKeyType,omitempty"`

	// The subject of the currently logged in user
	// +optional
	Subject string `json:"subject,omitempty"`

	// UID is the user uid
	// +optional
	UID string `json:"uid,omitempty"`

	// The groups of the currently logged in user
	// +optional
	Groups []string `json:"groups,omitempty"`

	// ChatAuthToken is the token used to authenticate with the in-product chat widget in the UI
	// +optional
	ChatAuthToken string `json:"chatAuthToken,omitempty"`

	// InstanceID is the loft instance id
	// +optional
	InstanceID string `json:"instanceID,omitempty"`

	// ProjectNamespacePrefix is the prefix used to name project namespaces after defaulting has been applied
	// +optional
	ProjectNamespacePrefix *string `json:"projectNamespacePrefix,omitempty"`
}

type UserInfo struct {
	storagev1.EntityInfo `json:",inline"`

	// Teams are the teams the user is part of
	// +optional
	Teams []*storagev1.EntityInfo `json:"teams,omitempty"`
}
