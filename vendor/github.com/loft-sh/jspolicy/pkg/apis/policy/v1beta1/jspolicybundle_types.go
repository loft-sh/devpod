package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JsPolicyBundle holds the bundled payload
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type JsPolicyBundle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JsPolicyBundleSpec   `json:"spec,omitempty"`
	Status JsPolicyBundleStatus `json:"status,omitempty"`
}

type JsPolicyBundleSpec struct {
	// Bundle holds the bundled payload (including dependencies and minified javascript code)
	// +optional
	Bundle []byte `json:"bundle,omitempty"`
}

type JsPolicyBundleStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JsPolicyBundleList contains a list of JsPolicyBundle
type JsPolicyBundleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JsPolicyBundle `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JsPolicyBundle{}, &JsPolicyBundleList{})
}
