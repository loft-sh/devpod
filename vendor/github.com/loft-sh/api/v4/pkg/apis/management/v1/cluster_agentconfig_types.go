package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterAgentConfig holds the loft agent configuration
// +subresource-request
type ClusterAgentConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	ClusterAgentConfigCommon `json:",inline"`
}

type ClusterAgentConfigCommon struct {
	// Cluster is the cluster the agent is running in.
	// +optional
	Cluster string `json:"cluster,omitempty"`

	// Audit holds the agent audit config
	// +optional
	Audit *AgentAuditConfig `json:"audit,omitempty"`

	// DefaultImageRegistry defines if we should prefix the virtual cluster image
	// +optional
	DefaultImageRegistry string `json:"defaultImageRegistry,omitempty"`

	// TokenCaCert is the certificate authority the Loft tokens will
	// be signed with
	// +optional
	TokenCaCert []byte `json:"tokenCaCert,omitempty"`

	// LoftHost defines the host for the agent's loft instance
	// +optional
	LoftHost string `json:"loftHost,omitempty"`

	// ProjectNamespacePrefix holds the prefix for loft project namespaces
	// +optional
	ProjectNamespacePrefix string `json:"projectNamespacePrefix,omitempty"`

	// LoftInstanceID defines the instance id from the loft instance
	// +optional
	LoftInstanceID string `json:"loftInstanceID,omitempty"`

	// AnalyticsSpec holds info needed for the agent to send analytics data to the analytics backend.
	AnalyticsSpec AgentAnalyticsSpec `json:"analyticsSpec"`
}

type AgentAuditConfig struct {
	// If audit is enabled and incoming api requests will be logged based on the supplied policy.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// If true, the agent will not send back any audit logs to Loft itself.
	// +optional
	DisableAgentSyncBack bool `json:"disableAgentSyncBack,omitempty"`

	// Level is an optional log level for audit logs. Cannot be used together with policy
	// +optional
	Level int `json:"level,omitempty"`

	// The audit policy to use and log requests. By default loft will not log anything
	// +optional
	Policy AuditPolicy `json:"policy,omitempty"`

	// The path where to save the audit log files. This is required if audit is enabled. Backup log files will
	// be retained in the same directory.
	// +optional
	Path string `json:"path,omitempty"`

	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	// +optional
	MaxAge int `json:"maxAge,omitempty"`

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	// +optional
	MaxBackups int `json:"maxBackups,omitempty"`

	// MaxSize is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	// +optional
	MaxSize int `json:"maxSize,omitempty"`

	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	// +optional
	Compress bool `json:"compress,omitempty"`
}

// AgentAnalyticsSpec holds info the agent can use to send analytics data to the analytics backend.
type AgentAnalyticsSpec struct {
	AnalyticsEndpoint string `json:"analyticsEndpoint,omitempty"`
}
