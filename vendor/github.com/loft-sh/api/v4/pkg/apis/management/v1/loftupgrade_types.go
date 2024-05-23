package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LoftUpgrade holds the upgrade information
// +k8s:openapi-gen=true
// +resource:path=loftupgrades,rest=LoftUpgradeREST
type LoftUpgrade struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LoftUpgradeSpec   `json:"spec,omitempty"`
	Status LoftUpgradeStatus `json:"status,omitempty"`
}

type LoftUpgradeSpec struct {
	// If specified, updated the release in the given namespace
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// If specified, uses this as release name
	// +optional
	Release string `json:"release,omitempty"`

	// +optional
	Version string `json:"version,omitempty"`
}

type LoftUpgradeStatus struct {
}
