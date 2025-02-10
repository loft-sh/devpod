package licenseapi

// ChatAuthCreateInput is the required input data for generating a hash for a user for in-product chat
// +k8s:deepcopy-gen=true
type ChatAuthCreateInput struct {
	*InstanceTokenAuth `hash:"-"`

	Provider string `json:"provider,omitempty"`
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Username string `json:"username,omitempty"`
}

// ChatAuthCreateOutput is the struct holding all information for chat auth
// generate user hash" requests.
// +k8s:deepcopy-gen=true
type ChatAuthCreateOutput struct {
	Hash string `json:"hash,omitempty"`
}
