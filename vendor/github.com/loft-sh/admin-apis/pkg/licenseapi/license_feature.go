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

	// +optional
	Description string `json:"description,omitempty"`

	// Status shows the status of the feature (see type FeatureStatus)
	// +optional
	Status string `json:"status,omitempty"`

	// Compatibility contains a series of semver compatibility constraints
	// +optional
	Compatibility string `json:"compatibility,omitempty"`

	// Labels contains a list of labels to be displayed for this feature (e.g. alpha, beta)
	// +optional
	Labels []string `json:"labels,omitempty"`
}
