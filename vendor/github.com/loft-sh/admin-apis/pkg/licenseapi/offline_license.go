package licenseapi

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type OfflineLicenseKeyClaims struct {
	License *License `json:"license,omitempty"`
}
