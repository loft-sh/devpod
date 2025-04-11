package devpod

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
)

type PlatformOptions struct {
	// Enabled is true if platform mode is enabled. Be careful with this option as this is only enabled
	// when executed on the platform side and not if a platform workspace is used locally.
	Enabled bool `json:"enabled,omitempty"`

	// DevPodWorkspaceInstance information
	InstanceName      string `json:"instanceName,omitempty"`
	InstanceProject   string `json:"instanceProject,omitempty"`
	InstanceNamespace string `json:"instanceNamespace,omitempty"`

	// connection options
	// AccessKey is used by the workspace daemon to authenticate itself
	AccessKey string `json:"accessKey,omitempty"`
	// UserAccessKey can be used as the workspace owner
	UserAccessKey string `json:"userAccessKey,omitempty"`
	WorkspaceHost string `json:"workspaceHost,omitempty"`
	PlatformHost  string `json:"platformHost,omitempty"`
	RunnerSocket  string `json:"runnerSocket,omitempty"`

	// environment template options
	EnvironmentTemplate        string                     `json:"environmentTemplate,omitempty"`
	EnvironmentTemplateVersion string                     `json:"environmentTemplateVersion,omitempty"`
	GitCloneStrategy           storagev1.GitCloneStrategy `json:"gitCloneStrategy,omitempty"`
	GitSkipLFS                 bool                       `json:"gitSkipLFS,omitempty"`

	// Kubernetes holds configuration for workspaces that need information about their kubernetes environment, i.e.
	// the ones running in virtual clusters or spaces
	Kubernetes *Kubernetes `json:"kubernetes,omitempty"`

	// user credentials are the credentials for the user
	UserCredentials    PlatformWorkspaceCredentials `json:"userCredentials,omitempty"`
	ProjectCredentials PlatformWorkspaceCredentials `json:"projectCredentials,omitempty"`

	// Remote builds
	Build *PlatformBuildOptions `json:"build,omitempty"`
}

type PlatformWorkspaceCredentials struct {
	GitUser  string                       `json:"gitUser,omitempty"`
	GitEmail string                       `json:"gitEmail,omitempty"`
	GitHttp  []PlatformGitHttpCredentials `json:"gitHttp,omitempty"`
	GitSsh   []PlatformGitSshCredentials  `json:"gitSsh,omitempty"`
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

type PlatformBuildOptions struct {
	Repository    string `json:"repository,omitempty"`
	RemoteAddress string `json:"remoteAddress,omitempty"`

	// mTLS
	CertCA  string `json:"certCa,omitempty"`
	CertKey string `json:"certKey,omitempty"`
	Cert    string `json:"cert,omitempty"`
}

type Kubernetes struct {
	SpaceName          string `json:"spaceName,omitempty"`
	VirtualClusterName string `json:"virtualClusterName,omitempty"`
	Namespace          string `json:"namespace,omitempty"`
}
