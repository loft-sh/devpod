package v1

import (
	auditv1 "github.com/loft-sh/api/v4/pkg/apis/audit/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AgentAuditEvent holds an event
// +k8s:openapi-gen=true
// +resource:path=agentauditevents,rest=AgentAuditEventsREST
type AgentAuditEvent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentAuditEventSpec   `json:"spec,omitempty"`
	Status AgentAuditEventStatus `json:"status,omitempty"`
}

// AgentAuditEventSpec holds the specification
type AgentAuditEventSpec struct {
	// Events are the events the agent has recorded
	// +optional
	Events []*auditv1.Event `json:"events,omitempty"`
}

// AgentAuditEventStatus holds the status
type AgentAuditEventStatus struct {
}
