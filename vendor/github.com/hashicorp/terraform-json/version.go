package tfjson

// VersionOutput represents output from the version -json command
// added in v0.13
type VersionOutput struct {
	Version            string            `json:"terraform_version"`
	Revision           string            `json:"terraform_revision"`
	Platform           string            `json:"platform,omitempty"`
	ProviderSelections map[string]string `json:"provider_selections"`
	Outdated           bool              `json:"terraform_outdated"`
}
