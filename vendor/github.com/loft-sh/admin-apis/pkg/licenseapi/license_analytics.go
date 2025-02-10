package licenseapi

// Analytics is a struct that represents the analytics server and the requests that should be sent
// to it. This information is sent to Loft instances when they check in with the license server.
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Analytics struct {
	// Endpoint is the endpoint for the analytics server.
	Endpoint string `json:"endpoint,omitempty"`
	// Requests is a slice of requested resources to return analytics for.
	// +optional
	Requests []Request `json:"requests,omitempty"`
}

// Request represents a request analytics information for an apigroup/resource and a list of verb actions for that
// resource.
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Request struct {
	// Group is the api group.
	// +optional
	Group string `json:"group,omitempty"`
	// Resource is the resource name for the request.
	// +optional
	Resource string `json:"resource,omitempty"`
	// Verbs is the list of verbs for the request.
	// +optional
	Verbs []string `json:"verbs,omitempty"`
}
