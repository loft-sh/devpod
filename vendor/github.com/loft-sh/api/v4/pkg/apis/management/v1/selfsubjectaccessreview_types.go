package v1

import (
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// User holds the user information
// +k8s:openapi-gen=true
// +resource:path=selfsubjectaccessreviews,rest=SelfSubjectAccessReviewREST
type SelfSubjectAccessReview struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SelfSubjectAccessReviewSpec   `json:"spec,omitempty"`
	Status SelfSubjectAccessReviewStatus `json:"status,omitempty"`
}

type SelfSubjectAccessReviewSpec struct {
	authorizationv1.SelfSubjectAccessReviewSpec `json:",inline"`
}

type SelfSubjectAccessReviewStatus struct {
	authorizationv1.SubjectAccessReviewStatus `json:",inline"`
}
