package v1

import (
	auditv1 "github.com/loft-sh/api/v4/pkg/apis/audit/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Event holds an event
// +k8s:openapi-gen=true
// +resource:path=events,rest=EventREST
type Event struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventSpec   `json:"spec,omitempty"`
	Status EventStatus `json:"status,omitempty"`
}

// EventSpec holds the specification
type EventSpec struct {
}

// EventStatus holds the status, which is the parsed raw config
type EventStatus struct {
	auditv1.Event `json:",inline"`
}
