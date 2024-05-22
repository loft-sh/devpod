package v1

import (
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// Level defines the amount of information logged during auditing
type Level string

// Valid audit levels
const (
	// LevelNone disables auditing
	LevelNone Level = "None"
	// LevelMetadata provides the basic level of auditing.
	LevelMetadata Level = "Metadata"
	// LevelRequest provides Metadata level of auditing, and additionally
	// logs the request object (does not apply for non-resource requests).
	LevelRequest Level = "Request"
	// LevelRequestResponse provides Request level of auditing, and additionally
	// logs the response object (does not apply for non-resource requests).
	LevelRequestResponse Level = "RequestResponse"
)

// RequestTarget defines the target of an incoming request
type RequestTarget string

// Valid request targets
const (
	// RequestTargetManagement specifies a loft management api request
	RequestTargetManagement RequestTarget = "Management"
	// RequestTargetCluster specifies a connected kubernetes cluster request
	RequestTargetCluster RequestTarget = "Cluster"
	// RequestTargetVCluster specifies a virtual kubernetes cluster request
	RequestTargetVCluster RequestTarget = "VCluster"
	// RequestTargetProjectSpace specifies a project space request
	RequestTargetProjectSpace RequestTarget = "ProjectSpace"
	// RequestTargetProjectVCluster specifies a project vcluster request
	RequestTargetProjectVCluster RequestTarget = "ProjectVCluster"
)

func ordLevel(l Level) int {
	switch l {
	case LevelMetadata:
		return 1
	case LevelRequest:
		return 2
	case LevelRequestResponse:
		return 3
	case LevelNone:
		return 0
	default:
		return 0
	}
}

func (a Level) Less(b Level) bool {
	return ordLevel(a) < ordLevel(b)
}

func (a Level) GreaterOrEqual(b Level) bool {
	return ordLevel(a) >= ordLevel(b)
}

// Stage defines the stages in request handling that audit events may be generated.
type Stage string

// Valid audit stages.
const (
	// The stage for events generated as soon as the audit handler receives the request, and before it
	// is delegated down the handler chain.
	StageRequestReceived Stage = "RequestReceived"
	// The stage for events generated once the response headers are sent, but before the response body
	// is sent. This stage is only generated for long-running requests (e.g. watch).
	StageResponseStarted Stage = "ResponseStarted"
	// The stage for events generated once the response body has been completed, and no more bytes
	// will be sent.
	StageResponseComplete Stage = "ResponseComplete"
	// The stage for events generated when a panic occurred.
	StagePanic Stage = "Panic"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Event holds the event information
// +k8s:openapi-gen=true
type Event struct {
	metav1.TypeMeta `json:",inline"`

	// AuditLevel at which event was generated
	Level Level `json:"level" protobuf:"bytes,1,opt,name=level,casttype=Level"`

	// Unique audit ID, generated for each request.
	AuditID types.UID `json:"auditID" protobuf:"bytes,2,opt,name=auditID,casttype=k8s.io/apimachinery/pkg/types.UID"`
	// Stage of the request handling when this event instance was generated.
	Stage Stage `json:"stage" protobuf:"bytes,3,opt,name=stage,casttype=Stage"`

	// RequestURI is the request URI as sent by the client to a server.
	RequestURI string `json:"requestURI" protobuf:"bytes,4,opt,name=requestURI"`
	// Verb is the kubernetes verb associated with the request.
	// For non-resource requests, this is the lower-cased HTTP method.
	Verb string `json:"verb" protobuf:"bytes,5,opt,name=verb"`
	// Authenticated user information.
	User authenticationv1.UserInfo `json:"user" protobuf:"bytes,6,opt,name=user"`
	// Impersonated user information.
	// +optional
	ImpersonatedUser *authenticationv1.UserInfo `json:"impersonatedUser,omitempty" protobuf:"bytes,7,opt,name=impersonatedUser"`
	// Source IPs, from where the request originated and intermediate proxies.
	// +optional
	SourceIPs []string `json:"sourceIPs,omitempty" protobuf:"bytes,8,rep,name=sourceIPs"`
	// UserAgent records the user agent string reported by the client.
	// Note that the UserAgent is provided by the client, and must not be trusted.
	// +optional
	UserAgent string `json:"userAgent,omitempty" protobuf:"bytes,16,opt,name=userAgent"`
	// Object reference this request is targeted at.
	// Does not apply for List-type requests, or non-resource requests.
	// +optional
	ObjectRef *ObjectReference `json:"objectRef,omitempty" protobuf:"bytes,9,opt,name=objectRef"`
	// The response status.
	// For successful and non-successful responses, this will only include the Code and StatusSuccess.
	// For panic type error responses, this will be auto-populated with the error Message.
	// +optional
	ResponseStatus *metav1.Status `json:"responseStatus,omitempty" protobuf:"bytes,10,opt,name=responseStatus"`

	// API object from the request, in JSON format. The RequestObject is recorded as-is in the request
	// (possibly re-encoded as JSON), prior to version conversion, defaulting, admission or
	// merging. It is an external versioned object type, and may not be a valid object on its own.
	// Omitted for non-resource requests.  Only logged at Request Level and higher.
	// +optional
	RequestObject *runtime.Unknown `json:"requestObject,omitempty" protobuf:"bytes,11,opt,name=requestObject"`
	// API object returned in the response, in JSON. The ResponseObject is recorded after conversion
	// to the external type, and serialized as JSON.  Omitted for non-resource requests.  Only logged
	// at Response Level.
	// +optional
	ResponseObject *runtime.Unknown `json:"responseObject,omitempty" protobuf:"bytes,12,opt,name=responseObject"`
	// Time the request reached the apiserver.
	// +optional
	RequestReceivedTimestamp metav1.MicroTime `json:"requestReceivedTimestamp" protobuf:"bytes,13,opt,name=requestReceivedTimestamp"`
	// Time the request reached current audit stage.
	// +optional
	StageTimestamp metav1.MicroTime `json:"stageTimestamp" protobuf:"bytes,14,opt,name=stageTimestamp"`

	// Annotations is an unstructured key value map stored with an audit event that may be set by
	// plugins invoked in the request serving chain, including authentication, authorization and
	// admission plugins. Note that these annotations are for the audit event, and do not correspond
	// to the metadata.annotations of the submitted object. Keys should uniquely identify the informing
	// component to avoid name collisions (e.g. podsecuritypolicy.admission.k8s.io/policy). Values
	// should be short. Annotations are included in the Metadata level.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,15,rep,name=annotations"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EventList is a list of audit Events.
type EventList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Event `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// ObjectReference contains enough information to let you inspect or modify the referred object.
type ObjectReference struct {
	// +optional
	Resource string `json:"resource,omitempty" protobuf:"bytes,1,opt,name=resource"`
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,2,opt,name=namespace"`
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,3,opt,name=name"`
	// +optional
	UID types.UID `json:"uid,omitempty" protobuf:"bytes,4,opt,name=uid,casttype=k8s.io/apimachinery/pkg/types.UID"`
	// APIGroup is the name of the API group that contains the referred object.
	// The empty string represents the core API group.
	// +optional
	APIGroup string `json:"apiGroup,omitempty" protobuf:"bytes,5,opt,name=apiGroup"`
	// APIVersion is the version of the API group that contains the referred object.
	// +optional
	APIVersion string `json:"apiVersion,omitempty" protobuf:"bytes,6,opt,name=apiVersion"`
	// +optional
	ResourceVersion string `json:"resourceVersion,omitempty" protobuf:"bytes,7,opt,name=resourceVersion"`
	// +optional
	Subresource string `json:"subresource,omitempty" protobuf:"bytes,8,opt,name=subresource"`
}
