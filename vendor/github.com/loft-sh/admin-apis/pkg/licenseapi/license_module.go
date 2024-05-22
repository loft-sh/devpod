package licenseapi

// Module is a struct representing a module of the product
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Module struct {
	// Name of the module (ModuleName)
	Name string `json:"name"`

	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// +optional
	Status FeatureStatus `json:"status,omitempty"`

	Limits   []*Limit   `json:"limits,omitempty"`
	Features []*Feature `json:"features,omitempty"`
}
