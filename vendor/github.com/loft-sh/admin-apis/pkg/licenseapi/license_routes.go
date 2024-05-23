package licenseapi

// LicenseAPIRoutes contains all key routes of the license api
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type LicenseAPIRoutes struct {
	ChatAuth LicenseAPIRoute `json:"chatAuth,omitempty"`

	FeatureDetails LicenseAPIRoute `json:"featureDetails,omitempty"`
	FeatureSetup   LicenseAPIRoute `json:"featureSetup,omitempty"`
	FeaturePreview LicenseAPIRoute `json:"featurePreview,omitempty"`

	ModuleActivation LicenseAPIRoute `json:"moduleActivation,omitempty"`
	ModulePreview    LicenseAPIRoute `json:"modulePreview,omitempty"`

	Checkout LicenseAPIRoute `json:"checkout,omitempty"`
	Portal   LicenseAPIRoute `json:"portal,omitempty"`
}

// LicenseAPIRoute is a single route of the license api
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type LicenseAPIRoute struct {
	URL    string `json:"url,omitempty"`
	Method string `json:"method,omitempty"`

	// Tells the frontend whether to make a direct request or to make it via the backend (via generic license api request)
	// +optional
	Direct bool `json:"direct,omitempty"`
}
