package v1beta1

import (
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JsPolicy holds the webhook configuration
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type JsPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JsPolicySpec   `json:"spec,omitempty"`
	Status JsPolicyStatus `json:"status,omitempty"`
}

type JsPolicySpec struct {
	// Operations is the operations the admission hook cares about - CREATE, UPDATE, DELETE, CONNECT or *
	// for all of those operations and any future admission operations that are added.
	// If '*' is present, the length of the slice must be one.
	// Required.
	Operations []admissionregistrationv1.OperationType `json:"operations,omitempty" protobuf:"bytes,1,rep,name=operations,casttype=OperationType"`

	// Resources is a list of resources this rule applies to.
	//
	// For example:
	// 'pods' means pods.
	// 'pods/log' means the log subresource of pods.
	// '*' means all resources, but not subresources.
	// 'pods/*' means all subresources of pods.
	// '*/scale' means all scale subresources.
	// '*/*' means all resources and their subresources.
	//
	// If wildcard is present, the validation rule will ensure resources do not
	// overlap with each other.
	//
	// Depending on the enclosing object, subresources might not be allowed.
	// Required.
	Resources []string `json:"resources,omitempty" protobuf:"bytes,3,rep,name=resources"`

	// APIGroups is the API groups the resources belong to. '*' is all groups.
	// If '*' is present, the length of the slice must be one.
	// +optional
	APIGroups []string `json:"apiGroups,omitempty" protobuf:"bytes,1,rep,name=apiGroups"`

	// APIVersions is the API versions the resources belong to. '*' is all versions.
	// If '*' is present, the length of the slice must be one.
	// +optional
	APIVersions []string `json:"apiVersions,omitempty" protobuf:"bytes,2,rep,name=apiVersions"`

	// scope specifies the scope of this rule.
	// Valid values are "Cluster", "Namespaced", and "*"
	// "Cluster" means that only cluster-scoped resources will match this rule.
	// Namespace API objects are cluster-scoped.
	// "Namespaced" means that only namespaced resources will match this rule.
	// "*" means that there are no scope restrictions.
	// Subresources match the scope of their parent resource.
	// Default is "*".
	//
	// +optional
	Scope *admissionregistrationv1.ScopeType `json:"scope,omitempty" protobuf:"bytes,4,rep,name=scope"`

	// FailurePolicy defines how unrecognized errors from the admission endpoint are handled -
	// allowed values are Ignore or Fail. Defaults to Fail.
	// +optional
	FailurePolicy *admissionregistrationv1.FailurePolicyType `json:"failurePolicy,omitempty" protobuf:"bytes,4,opt,name=failurePolicy,casttype=FailurePolicyType"`

	// matchPolicy defines how the "rules" list is used to match incoming requests.
	// Allowed values are "Exact" or "Equivalent".
	//
	// - Exact: match a request only if it exactly matches a specified rule.
	// For example, if deployments can be modified via apps/v1, apps/v1beta1, and extensions/v1beta1,
	// but "rules" only included `apiGroups:["apps"], apiVersions:["v1"], resources: ["deployments"]`,
	// a request to apps/v1beta1 or extensions/v1beta1 would not be sent to the webhook.
	//
	// - Equivalent: match a request if modifies a resource listed in rules, even via another API group or version.
	// For example, if deployments can be modified via apps/v1, apps/v1beta1, and extensions/v1beta1,
	// and "rules" only included `apiGroups:["apps"], apiVersions:["v1"], resources: ["deployments"]`,
	// a request to apps/v1beta1 or extensions/v1beta1 would be converted to apps/v1 and sent to the webhook.
	//
	// Defaults to "Equivalent"
	// +optional
	MatchPolicy *admissionregistrationv1.MatchPolicyType `json:"matchPolicy,omitempty" protobuf:"bytes,9,opt,name=matchPolicy,casttype=MatchPolicyType"`

	// NamespaceSelector decides whether to run the webhook on an object based
	// on whether the namespace for that object matches the selector. If the
	// object itself is a namespace, the matching is performed on
	// object.metadata.labels. If the object is another cluster scoped resource,
	// it never skips the webhook.
	//
	// For example, to run the webhook on any objects whose namespace is not
	// associated with "runlevel" of "0" or "1";  you will set the selector as
	// follows:
	// "namespaceSelector": {
	//   "matchExpressions": [
	//     {
	//       "key": "runlevel",
	//       "operator": "NotIn",
	//       "values": [
	//         "0",
	//         "1"
	//       ]
	//     }
	//   ]
	// }
	//
	// If instead you want to only run the webhook on any objects whose
	// namespace is associated with the "environment" of "prod" or "staging";
	// you will set the selector as follows:
	// "namespaceSelector": {
	//   "matchExpressions": [
	//     {
	//       "key": "environment",
	//       "operator": "In",
	//       "values": [
	//         "prod",
	//         "staging"
	//       ]
	//     }
	//   ]
	// }
	//
	// See
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels
	// for more examples of label selectors.
	//
	// Default to the empty LabelSelector, which matches everything.
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty" protobuf:"bytes,5,opt,name=namespaceSelector"`

	// ObjectSelector decides whether to run the webhook based on if the
	// object has matching labels. objectSelector is evaluated against both
	// the oldObject and newObject that would be sent to the webhook, and
	// is considered to match if either object matches the selector. A null
	// object (oldObject in the case of create, or newObject in the case of
	// delete) or an object that cannot have labels (like a
	// DeploymentRollback or a PodProxyOptions object) is not considered to
	// match.
	// Use the object selector only if the webhook is opt-in, because end
	// users may skip the admission webhook by setting the labels.
	// Default to the empty LabelSelector, which matches everything.
	// +optional
	ObjectSelector *metav1.LabelSelector `json:"objectSelector,omitempty" protobuf:"bytes,10,opt,name=objectSelector"`

	// TimeoutSeconds specifies the timeout for this webhook. After the timeout passes,
	// the webhook call will be ignored or the API call will fail based on the
	// failure policy.
	// The timeout value must be between 1 and 30 seconds.
	// Default to 10 seconds.
	// +optional
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty" protobuf:"varint,7,opt,name=timeoutSeconds"`

	// Violation policy describes how violations should be handled. You can either specify deny (which is the default),
	// warn or dry.
	// +optional
	ViolationPolicy *ViolationPolicyType `json:"violationPolicy,omitempty"`

	// AuditPolicy defines if violations should be logged to the webhook status or not. By default, violations
	// will be logged to the CRD status.
	// +optional
	AuditPolicy *AuditPolicyType `json:"auditPolicy,omitempty"`

	// AuditLogSize defines how many violations should be logged in the status. Defaults to 10
	// +optional
	AuditLogSize *int32 `json:"auditLogSize,omitempty"`

	// Type defines what kind of policy the object represents. Valid values are Validating, Mutating and
	// Controller. Defaults to Validating.
	// +optional
	Type PolicyType `json:"type,omitempty"`

	// Dependencies is a map of npm modules this webhook should be bundled with
	// +optional
	Dependencies map[string]string `json:"dependencies,omitempty"`

	// JavaScript is the payload of the webhook that will be executed. If this is not defined,
	// jsPolicy expects the user to create a JsPolicyBundle for this policy.
	// +optional
	JavaScript string `json:"javascript,omitempty"`
}

type JsPolicyStatus struct {
	// Phase describes how the syncing status of the webhook is
	// +optional
	Phase WebhookPhase `json:"phase,omitempty"`

	// Reason holds the error in machine-readable language if the webhook is in a failed state
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes the error in human-readable language if the webhook is in a failed state
	// +optional
	Message string `json:"message,omitempty"`

	// Conditions holds several conditions the virtual cluster might be in
	// +optional
	Conditions Conditions `json:"conditions,omitempty"`

	// ObservedGeneration is the latest generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// BundleHash is used to determine if we have to re-bundle the javascript
	// +optional
	BundleHash string `json:"bundleHash,omitempty"`
}

// PolicyType is the type of a JsPolicy
type PolicyType string

const (
	// PolicyTypeValidating indicates that the JsPolicy should be a Validating webhook
	PolicyTypeValidating PolicyType = "Validating"
	// PolicyTypeMutating indicates that the JsPolicy should be a Mutating webhook
	PolicyTypeMutating PolicyType = "Mutating"
	// PolicyTypeController indicates that the JsPolicy should be a Kubernetes controller
	PolicyTypeController PolicyType = "Controller"
)

// ViolationPolicyType specify how to handle violations
type ViolationPolicyType string

const (
	// ViolationPolicyPolicyDeny indicates that the webhook should deny the request
	// if it violates the specified javascript rule.
	ViolationPolicyPolicyDeny ViolationPolicyType = "Deny"
	// ViolationPolicyPolicyWarn indicates that the webhook should warn the user that
	// the request violates the specified javascript rule.
	ViolationPolicyPolicyWarn ViolationPolicyType = "Warn"
	// ViolationPolicyPolicyDry indicates that the webhook should record the violation
	// but not deny or warn the user about it.
	ViolationPolicyPolicyDry ViolationPolicyType = "Dry"
	// ViolationPolicyPolicyController indicates that the violation was written by
	// a controller policy that did not do any action.
	ViolationPolicyPolicyController ViolationPolicyType = "Controller"
)

type AuditPolicyType string

const (
	AuditPolicyLog  AuditPolicyType = "Log"
	AuditPolicySkip AuditPolicyType = "Skip"
)

type WebhookPhase string

const (
	WebhookPhaseSynced WebhookPhase = "Synced"
	WebhookPhaseFailed WebhookPhase = "Failed"
)

// GetConditions returns the set of conditions for this object.
func (in *JsPolicy) GetConditions() Conditions {
	return in.Status.Conditions
}

// SetConditions sets the conditions on this object.
func (in *JsPolicy) SetConditions(conditions Conditions) {
	in.Status.Conditions = conditions
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JsPolicyList contains a list of JsPolicy
type JsPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JsPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JsPolicy{}, &JsPolicyList{})
}
