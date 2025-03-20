package licenseapi

// Feature contains information regarding to a feature
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Feature struct {
	// Name is the name of the feature (FeatureName)
	// This cannot be FeatureName because it needs to be downward compatible
	// e.g. older Loft version doesn't know a newer feature but it will still be received and still needs to be rendered in the license view
	Name string `json:"name"`

	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Preview represents whether the feature can be previewed if a user's license does not allow the feature
	// +optional
	Preview bool `json:"preview,omitempty"`

	// Status shows the status of the feature (see type FeatureStatus)
	// +optional
	Status string `json:"status,omitempty"`
}
