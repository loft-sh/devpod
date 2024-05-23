package licenseapi

// +k8s:deepcopy-gen=true
type DomainToken struct {
	URL string `json:"url"`
}
