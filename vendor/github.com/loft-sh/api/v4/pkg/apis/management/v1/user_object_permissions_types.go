package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type UserObjectPermissions struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	ObjectPermissions []ObjectPermission `json:"objectPermissions,omitempty"`
}

type ObjectPermission struct {
	ObjectName `json:",inline"`

	// Verbs is a list of actions allowed by the user on the object. '*' represents all verbs
	Verbs []string `json:"verbs" protobuf:"bytes,1,rep,name=verbs"`
}
