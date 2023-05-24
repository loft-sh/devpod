package provider

import (
	"github.com/loft-sh/devpod/pkg/types"
)

const (
	CommandEnv = "COMMAND"
)

type ProviderConfig struct {
	// Name is the name of the provider
	Name string `json:"name,omitempty"`

	// Version is the provider version
	Version string `json:"version,omitempty"`

	// Icon holds an image URL that will be displayed
	Icon string `json:"icon,omitempty"`

	// Home holds the provider home URL
	Home string `json:"home,omitempty"`

	// Source is the source the provider was loaded from
	Source ProviderSource `json:"source,omitempty"`

	// Description is the provider description
	Description string `json:"description,omitempty"`

	// OptionGroups holds information how to display options
	OptionGroups []ProviderOptionGroup `json:"optionGroups,omitempty"`

	// Options are the provider options.
	Options map[string]*ProviderOption `json:"options,omitempty"`

	// Agent allows you to override agent configuration
	Agent ProviderAgentConfig `json:"agent,omitempty"`

	// Exec holds the provider commands
	Exec ProviderCommands `json:"exec,omitempty"`

	// Binaries is an optional field to specify a binary to execute the commands
	Binaries map[string][]*ProviderBinary `json:"binaries,omitempty"`
}

type ProviderOptionGroup struct {
	// Name is the display name of the option group
	Name string `json:"name,omitempty"`

	// Options are the options that belong to this group
	Options []string `json:"options,omitempty"`

	// DefaultVisible defines if the option group should be visible by default
	DefaultVisible bool `json:"defaultVisible,omitempty"`
}

type ProviderSource struct {
	// Internal means provider was received internally
	Internal bool `json:"internal,omitempty"`

	// Github source for the provider
	Github string `json:"github,omitempty"`

	// File source for the provider
	File string `json:"file,omitempty"`

	// URL where the provider was downloaded from
	URL string `json:"url,omitempty"`
}

type ProviderAgentConfig struct {
	// Local defines if DevPod is running locally
	Local types.StrBool `json:"local,omitempty"`

	// Path is the binary path inside the server devpod will expect the agent binary
	Path string `json:"path,omitempty"`

	// DataPath is the agent path where data is stored
	DataPath string `json:"dataPath,omitempty"`

	// DownloadURL is the base url where to download the agent from
	DownloadURL string `json:"downloadURL,omitempty"`

	// Timeout is the timeout in minutes to wait until the agent tries
	// to turn of the server.
	Timeout string `json:"inactivityTimeout,omitempty"`

	// ContainerTimeout is the timeout in minutes to wait until the agent tries
	// to delete the container.
	ContainerTimeout string `json:"containerInactivityTimeout,omitempty"`

	// InjectGitCredentials signals DevPod if git credentials should get synced into
	// the remote machine for cloning the repository.
	InjectGitCredentials types.StrBool `json:"injectGitCredentials,omitempty"`

	// InjectDockerCredentials signals DevPod if docker credentials should get synced
	// into the remote machine for pulling and pushing images.
	InjectDockerCredentials types.StrBool `json:"injectDockerCredentials,omitempty"`

	// Exec commands that can be used on the remote
	Exec ProviderAgentConfigExec `json:"exec,omitempty"`

	// Binaries is an optional field to specify a binary to execute the commands
	Binaries map[string][]*ProviderBinary `json:"binaries,omitempty"`

	// Driver is the driver to use for deploying the devcontainer. Currently supports
	// docker (default) or kubernetes (experimental)
	Driver string `json:"driver,omitempty"`

	// Kubernetes holds kubernetes specific configuration
	Kubernetes ProviderKubernetesDriverConfig `json:"kubernetes,omitempty"`

	// Docker holds docker specific configuration
	Docker ProviderDockerDriverConfig `json:"docker,omitempty"`
}

const (
	DockerDriver     = "docker"
	KubernetesDriver = "kubernetes"
)

type ProviderKubernetesDriverConfig struct {
	// Path where to find the kubectl binary, defaults to 'kubectl'
	Path string `json:"path,omitempty"`

	// Namespace is the Kubernetes namespace to use
	Namespace string `json:"namespace,omitempty"`

	// CreateNamespace specifies if DevPod should try to create the namespace
	CreateNamespace types.StrBool `json:"createNamespace,omitempty"`

	// Context is the context to use
	Context string `json:"context,omitempty"`

	// Config is the path to the kube config to use
	Config string `json:"config,omitempty"`

	// ClusterRole defines a role binding with the given cluster role
	// DevPod should create.
	ClusterRole string `json:"clusterRole,omitempty"`

	// ServiceAccount is the service account to use
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// BuildRepository defines the repository to push builds. If empty,
	// DevPod will not try to build any images at all.
	BuildRepository string `json:"buildRepository,omitempty"`

	// BuildkitImage is the build kit image to use
	BuildkitImage string `json:"buildkitImage,omitempty"`

	// BuildkitPrivileged signals if pod should be ran in privileged mode
	BuildkitPrivileged types.StrBool `json:"buildkitPrivileged,omitempty"`

	// HelperImage is used to find out cluster architecture and copy files
	HelperImage string `json:"helperImage,omitempty"`

	// PersistentVolumeSize is the size of the persistent volume in GB
	PersistentVolumeSize string `json:"persistentVolumeSize,omitempty"`
}

type ProviderDockerDriverConfig struct {
	// Path where to find the docker binary, defaults to 'docker'
	Path string `json:"path,omitempty"`

	// If false, DevPod will not try to install docker into the machine.
	Install types.StrBool `json:"install,omitempty"`

	// Environment variables to set when running docker commands
	Env map[string]string `json:"envs,omitempty"`
}

type ProviderAgentConfigExec struct {
	// Shutdown is the remote command to run when the remote machine
	// should shutdown.
	Shutdown types.StrArray `json:"shutdown,omitempty"`
}

type ProviderBinary struct {
	// The current OS
	OS string `json:"os,omitempty"`

	// The current Arch
	Arch string `json:"arch,omitempty"`

	// Checksum is the sha256 hash of the binary
	Checksum string `json:"checksum,omitempty"`

	// Path is the binary url to download from or relative path to use
	Path string `json:"path,omitempty"`

	// ArchivePath is the path within the archive to extract
	ArchivePath string `json:"archivePath,omitempty"`

	// Name is the name of the binary to store locally
	Name string `json:"name,omitempty"`
}

type ProviderCommands struct {
	// Init is run directly after `devpod provider use`
	Init types.StrArray `json:"init,omitempty"`

	// Command executes a command on the server
	Command types.StrArray `json:"command,omitempty"`

	// Create creates a new server
	Create types.StrArray `json:"create,omitempty"`

	// Delete destroys a server
	Delete types.StrArray `json:"delete,omitempty"`

	// Start starts a stopped server
	Start types.StrArray `json:"start,omitempty"`

	// Stop stops a running server
	Stop types.StrArray `json:"stop,omitempty"`

	// Status retrieves the server status
	Status types.StrArray `json:"status,omitempty"`
}

type ProviderOption struct {
	// A description of the option displayed to the user by a supporting tool.
	Description string `json:"description,omitempty"`

	// If required is true and the user doesn't supply a value, devpod will ask the user
	Required bool `json:"required,omitempty"`

	// If true, will not show the value to the user
	Password bool `json:"password,omitempty"`

	// Type is the provider option type. Can be one of: string, duration, number or boolean. Defaults to string
	Type string `json:"type,omitempty"`

	// ValidationPattern is a regex pattern to validate the value
	ValidationPattern string `json:"validationPattern,omitempty"`

	// ValidationMessage is the message that appears if the user enters an invalid option
	ValidationMessage string `json:"validationMessage,omitempty"`

	// Suggestions are suggestions to show in the DevPod UI for this option
	Suggestions []string `json:"suggestions,omitempty"`

	// Allowed values for this option.
	Enum []string `json:"enum,omitempty"`

	// Hidden specifies if the option should be hidden
	Hidden bool `json:"hidden,omitempty"`

	// Local means the variable is not resolved immediately and instead later when the workspace / machine was created.
	Local bool `json:"local,omitempty"`

	// Global means the variable is stored globally. By default, option values will be
	// saved per machine or workspace instead.
	Global bool `json:"global,omitempty"`

	// Default value if the user omits this option from their configuration.
	Default string `json:"default,omitempty"`

	// Cache is the duration to cache the value before rerunning the command
	Cache string `json:"cache,omitempty"`

	// Command is the command to run to specify an option
	Command string `json:"command,omitempty"`
}

func (c *ProviderConfig) IsMachineProvider() bool {
	return len(c.Exec.Create) > 0
}
