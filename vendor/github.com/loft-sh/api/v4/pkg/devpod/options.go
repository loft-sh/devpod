package devpod

type CloneOptions struct {
	// Repository is the repository to clone
	Repository string `json:"repository,omitempty"`

	// Branch is the branch to use
	Branch string `json:"branch,omitempty"`

	// Commit is the commit SHA to checkout
	Commit string `json:"commit,omitempty"`

	// PRReference is the pull request reference to checkout
	PRReference string `json:"prReference,omitempty"`

	// SubPath is the subpath in the repo to use
	SubPath string `json:"subPath,omitempty"`

	// CredentialsHelper is the credentials helper to use for the clone
	CredentialsHelper string `json:"credentialsHelper,omitempty"`

	// ExtraEnv is the extra environment variables to use for the clone
	ExtraEnv []string `json:"extraEnv,omitempty"`
}
