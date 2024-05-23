package licenseapi

// Trial represents a trial
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Trial struct {
	// ID is the unique id of this trial
	ID string `json:"id,omitempty"`

	// DisplayName is a display name for the trial
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Start is the unix timestamp stating when the trial was started
	// +optional
	Start *int64 `json:"start,omitempty"`

	// End is the unix timestamp stating when the trial will end or ended
	// +optional
	End int64 `json:"end,omitempty"`

	// Status is the status of this trial (TrialStatus)
	// +optional
	Status string `json:"status,omitempty"`

	// DowngradesTo states the name of the plan that is replacing the current one once the trial expires
	// If this is nil, then this plan just expires (i.e. the subscription may be canceled, paused, etc.)
	// +optional
	DowngradesTo *string `json:"downgradesTo,omitempty"`
}
