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

	// IconDark holds an image URL that will be displayed in dark mode
	IconDark string `json:"iconDark,omitempty"`

	// Home holds the provider home URL
	Home string `json:"home,omitempty"`

	// Source is the source the provider was loaded from
	Source ProviderSource `json:"source,omitempty"`

	// Description is the provider description
	Description string `json:"description,omitempty"`

	// OptionGroups holds information how to display options
	OptionGroups []ProviderOptionGroup `json:"optionGroups,omitempty"`

	// Options are the provider options.
	Options map[string]*types.Option `json:"options,omitempty"`

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

	// Raw is the exact string we used to load the provider
	Raw string `json:"raw,omitempty"`
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

	// Dockerless holds custom dockerless configuration
	Dockerless ProviderDockerlessOptions `json:"dockerless,omitempty"`

	// Driver is the driver to use for deploying the devcontainer. Currently supports
	// docker (default) or kubernetes (experimental)
	Driver string `json:"driver,omitempty"`

	// Docker holds docker specific configuration
	Docker ProviderDockerDriverConfig `json:"docker,omitempty"`

	// Custom holds custom driver specific configuration
	Custom ProviderCustomDriverConfig `json:"custom,omitempty"`

	// Kubernetes holds kubernetes specific configuration
	Kubernetes ProviderKubernetesDriverConfig `json:"kubernetes,omitempty"`
}

type ProviderDockerlessOptions struct {
	// Disabled signals if dockerless building is disabled
	Disabled types.StrBool `json:"disabled,omitempty"`

	// Image is the image of the dockerless container to start
	Image string `json:"image,omitempty"`

	// IgnorePaths are additional ignore paths that should be ignored during deletion
	IgnorePaths string `json:"ignorePaths,omitempty"`

	// Registry to use as remote cache
	RegistryCache string `json:"registryCache,omitempty"`

	// DisableDockerCredentials prevents docker credentials from getting injected
	DisableDockerCredentials types.StrBool `json:"disableDockerCredentials,omitempty"`
}

func (a ProviderAgentConfig) IsDockerDriver() bool {
	return a.Driver == "" || a.Driver == DockerDriver
}

const (
	DockerDriver     = "docker"
	KubernetesDriver = "kubernetes"
	CustomDriver     = "custom"
)

type ProviderCustomDriverConfig struct {
	// FindDevContainer is used to find an existing devcontainer
	FindDevContainer types.StrArray `json:"findDevContainer,omitempty"`

	// CommandDevContainer is used to execute a command in the devcontainer
	CommandDevContainer types.StrArray `json:"commandDevContainer,omitempty"`

	// TargetArchitecture is used to find out the target architecture
	TargetArchitecture types.StrArray `json:"targetArchitecture,omitempty"`

	// RunDevContainer is used to actually run the devcontainer
	RunDevContainer types.StrArray `json:"runDevContainer,omitempty"`

	// StartDevContainer is used to start the devcontainer
	StartDevContainer types.StrArray `json:"startDevContainer,omitempty"`

	// StopDevContainer is used to stop the devcontainer
	StopDevContainer types.StrArray `json:"stopDevContainer,omitempty"`

	// DeleteDevContainer is used to delete the devcontainer
	DeleteDevContainer types.StrArray `json:"deleteDevContainer,omitempty"`

	// CanReprovision returns true if the driver can reprovision the devcontainer
	CanReprovision types.StrBool `json:"canReprovision,omitempty"`

	// GetDevContainerLogs returns the logs of the devcontainer
	GetDevContainerLogs types.StrArray `json:"getDevContainerLogs,omitempty"`
}

type ProviderDockerDriverConfig struct {
	// Path where to find the docker binary, defaults to 'docker'
	Path string `json:"path,omitempty"`

	// If false, DevPod will not try to install docker into the machine.
	Install types.StrBool `json:"install,omitempty"`

	// Builder to use with docker
	Builder string `json:"builder,omitempty"`

	// Environment variables to set when running docker commands
	Env map[string]string `json:"env,omitempty"`
}

type ProviderKubernetesDriverConfig struct {
	KubernetesContext   string `json:"kubernetesContext,omitempty"`
	KubernetesConfig    string `json:"kubernetesConfig,omitempty"`
	KubernetesNamespace string `json:"kubernetesNamespace,omitempty"`
	PodTimeout          string `json:"podTimeout,omitempty"`

	KubernetesPullSecretsEnabled string `json:"kubernetesPullSecretsEnabled,omitempty"`
	CreateNamespace              string `json:"createNamespace,omitempty"`
	ClusterRole                  string `json:"clusterRole,omitempty"`
	ServiceAccount               string `json:"serviceAccount,omitempty"`

	Architecture      string `json:"architecture,omitempty"`
	InactivityTimeout string `json:"inactivityTimeout,omitempty"`
	StorageClass      string `json:"storageClass,omitempty"`

	DiskSize             string `json:"diskSize,omitempty"`
	PvcAccessMode        string `json:"pvcAccessMode,omitempty"`
	PvcAnnotations       string `json:"pvcAnnotations,omitempty"`
	NodeSelector         string `json:"nodeSelector,omitempty"`
	Resources            string `json:"resources,omitempty"`
	WorkspaceVolumeMount string `json:"workspaceVolumeMount,omitempty"`

	PodManifestTemplate string `json:"podManifestTemplate,omitempty"`
	Labels              string `json:"labels,omitempty"`

	StrictSecurity string `json:"strictSecurity,omitempty"`
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

	// Proxy proxies commands
	Proxy *ProxyCommands `json:"proxy,omitempty"`

	// Daemon commands
	Daemon *DaemonCommands `json:"daemon,omitempty"`
}

type ProxyCommands struct {
	// Up proxies the up command
	Up types.StrArray `json:"up,omitempty"`

	// Stop proxies the stop command
	Stop types.StrArray `json:"stop,omitempty"`

	// Delete proxies the delete command
	Delete types.StrArray `json:"delete,omitempty"`

	// Ssh proxies the ssh command
	Ssh types.StrArray `json:"ssh,omitempty"`

	// Status proxies the status command
	Status types.StrArray `json:"status,omitempty"`

	// Health checks the health of the platform
	Health types.StrArray `json:"health,omitempty"`

	// Create creates entities associated with this provider
	Create CreateProxyCommands `json:"create,omitempty"`

	// Get gets entities associated with this provider
	Get GetProxyCommands `json:"get,omitempty"`

	// List lists all entities associated with this provider
	List ListProxyCommands `json:"list,omitempty"`

	// Watch lists all entities associated with this provider and then watches for changes
	Watch WatchProxyCommands `json:"watch,omitempty"`

	// Update updates entities associated with this provider
	Update UpdateProxyCommands `json:"update,omitempty"`
}

type ListProxyCommands struct {
	// Workspaces lists all workspaces
	Workspaces types.StrArray `json:"workspaces,omitempty"`

	// Projects lists all projects
	Projects types.StrArray `json:"projects,omitempty"`

	// Templates lists all templates in a project
	Templates types.StrArray `json:"templates,omitempty"`

	// Clusters lists all clusters and runners in a project
	Clusters types.StrArray `json:"clusters,omitempty"`
}

type WatchProxyCommands struct {
	// Workspaces watches all workspaces and updates on changes
	Workspaces types.StrArray `json:"workspaces,omitempty"`
}

type CreateProxyCommands struct {
	// Workspace creates a workspace instance
	Workspace types.StrArray `json:"workspace,omitempty"`
}

type GetProxyCommands struct {
	// Workspace gets a workspace instance
	Workspace types.StrArray `json:"workspace,omitempty"`

	// Self gets self for this provider
	Self types.StrArray `json:"self,omitempty"`

	// Version gets the for this pro instance
	Version types.StrArray `json:"version,omitempty"`
}

type UpdateProxyCommands struct {
	// Workspace updates a workspace instance
	Workspace types.StrArray `json:"workspace,omitempty"`
}

type DaemonCommands struct {
	// Start starts the daemon
	Start types.StrArray `json:"start,omitempty"`
	// Status gets the daemon status
	Status types.StrArray `json:"status,omitempty"`
}

type SubOptions struct {
	Options map[string]*types.Option `json:"options,omitempty"`
}

func (c *ProviderConfig) IsMachineProvider() bool {
	return len(c.Exec.Create) > 0
}

func (c *ProviderConfig) IsProxyProvider() bool {
	return c.Exec.Proxy != nil
}

func (c *ProviderConfig) HasHealthCheck() bool {
	return c.Exec.Proxy != nil && len(c.Exec.Proxy.Health) > 0
}

func (c *ProviderConfig) IsDaemonProvider() bool {
	return c.Exec.Daemon != nil
}
