package licenseapi

// BlockRequest tells the instance to block certain requests due to overages (limit exceeded)
// the license server.
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type BlockRequest struct {
	Request `json:",inline"`

	Overage *ResourceCount `json:"overage,omitempty"`
}
