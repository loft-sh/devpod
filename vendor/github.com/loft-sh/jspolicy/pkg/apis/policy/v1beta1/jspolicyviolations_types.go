package v1beta1

import (
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JsPolicyViolations holds the webhook configuration
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type JsPolicyViolations struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JsPolicyViolationsSpec   `json:"spec,omitempty"`
	Status JsPolicyViolationsStatus `json:"status,omitempty"`
}

type JsPolicyViolationsSpec struct {
}

type JsPolicyViolationsStatus struct {
	// Violations is an array of violations that were recorded by the webhook
	// +optional
	Violations []PolicyViolation `json:"violations,omitempty"`
}

type PolicyViolation struct {
	// Action holds the the action type the webhook reacted with
	// +optional
	Action string `json:"action,omitempty"`

	// Code is the error code that was returned to the client
	// +optional
	Code int32 `json:"code,omitempty"`

	// Reason is the error reason that was returned to the client
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message holds the message that was sent to the client
	// +optional
	Message string `json:"message,omitempty"`

	// The request this violation is about
	// +optional
	RequestInfo *RequestInfo `json:"requestInfo,omitempty"`

	// The user that sent the request
	// +optional
	UserInfo *UserInfo `json:"userInfo,omitempty"`

	// The timestamp when this violation has occurred
	// +optional
	Timestamp metav1.Time `json:"timestamp,omitempty"`
}

type RequestInfo struct {
	// Name is the name of the object as presented in the request. On a CREATE operation, the client may omit name and
	// rely on the server to generate the name. If that is the case, this field will contain an empty string.
	// +optional
	Name string `json:"name,omitempty"`
	// Namespace is the namespace associated with the request (if any).
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// Kind is the type of object being submitted (for example, Pod or Deployment)
	// +optional
	Kind string `json:"kind,omitempty"`
	// Kind is the type of object being submitted (for example, Pod or Deployment)
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
	// Operation is the operation being performed. This may be different than the operation
	// requested. e.g. a patch can result in either a CREATE or UPDATE Operation.
	// +optional
	Operation admissionv1.Operation `json:"operation,omitempty"`
}

type UserInfo struct {
	// The name that uniquely identifies this user among all active users.
	// +optional
	Username string `json:"username,omitempty" protobuf:"bytes,1,opt,name=username"`
	// A unique value that identifies this user across time. If this user is
	// deleted and another user by the same name is added, they will have
	// different UIDs.
	// +optional
	UID string `json:"uid,omitempty" protobuf:"bytes,2,opt,name=uid"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JsPolicyViolationsList contains a list of JsPolicyViolations
type JsPolicyViolationsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JsPolicyViolations `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JsPolicyViolations{}, &JsPolicyViolationsList{})
}
