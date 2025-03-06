package provider

type PlatformOptions struct {
	// Enabled is true if platform mode is enabled. Be careful with this option as this is only enabled
	// when executed on the platform side and not if a platform workspace is used locally.
	Enabled bool `json:"enabled,omitempty"`

	// connection options
	AccessKey     string `json:"accessKey,omitempty"`
	WorkspaceHost string `json:"workspaceHost,omitempty"`
	PlatformHost  string `json:"platformHost,omitempty"`

	// environment template options
	EnvironmentTemplate        string `json:"environmentTemplate,omitempty"`
	EnvironmentTemplateVersion string `json:"environmentTemplateVersion,omitempty"`

	// credentials for the workspace
	Credentials PlatformWorkspaceCredentials `json:"credentials,omitempty"`

	// Remote builds
	BuildRegistry  string `json:"registry,omitempty"`
	BuilderAddress string `json:"builderAddress,omitempty"`
}

type PlatformWorkspaceCredentials struct {
	GitHttp []PlatformGitHttpCredentials `json:"gitHttp,omitempty"`
	GitSsh  []PlatformGitSshCredentials  `json:"gitSsh,omitempty"`
	Docker  []PlatformDockerCredentials  `json:"docker,omitempty"`
}

type PlatformGitHttpCredentials struct {
	Host     string `json:"host,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	Path     string `json:"path,omitempty"`
}

type PlatformGitSshCredentials struct {
	Key string `json:"key,omitempty"`
}

type PlatformDockerCredentials struct {
	Host     string `json:"host,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}
