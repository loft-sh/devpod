package provider

import (
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/config"
	devcontainerconfig "github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/flags"
	"github.com/loft-sh/devpod/pkg/git"
	"github.com/loft-sh/devpod/pkg/types"
)

var (
	WorkspaceSourceGit       = "git:"
	WorkspaceSourceLocal     = "local:"
	WorkspaceSourceImage     = "image:"
	WorkspaceSourceContainer = "container:"
)

type Workspace struct {
	// ID is the workspace id to use
	ID string `json:"id,omitempty"`

	// UID is used to identify this specific workspace
	UID string `json:"uid,omitempty"`

	// Picture is the project social media image
	Picture string `json:"picture,omitempty"`

	// Provider is the provider used to create this workspace
	Provider WorkspaceProviderConfig `json:"provider,omitempty"`

	// Machine is the machine to use for this workspace
	Machine WorkspaceMachineConfig `json:"machine,omitempty"`

	// IDE holds IDE specific settings
	IDE WorkspaceIDEConfig `json:"ide,omitempty"`

	// Source is the source where this workspace will be created from
	Source WorkspaceSource `json:"source,omitempty"`

	// DevContainerImage is the container image to use, overriding whatever is in the devcontainer.json
	DevContainerImage string `json:"devContainerImage,omitempty"`

	// DevContainerPath is the relative path where the devcontainer.json is located.
	DevContainerPath string `json:"devContainerPath,omitempty"`

	// DevContainerConfig holds the config for the devcontainer.json.
	DevContainerConfig *devcontainerconfig.DevContainerConfig `json:"devContainerConfig,omitempty"`

	// CreationTimestamp is the timestamp when this workspace was created
	CreationTimestamp types.Time `json:"creationTimestamp,omitempty"`

	// LastUsedTimestamp holds the timestamp when this workspace was last accessed
	LastUsedTimestamp types.Time `json:"lastUsed,omitempty"`

	// Context is the context where this config file was loaded from
	Context string `json:"context,omitempty"`

	// Imported signals that this workspace was imported
	Imported bool `json:"imported,omitempty"`

	// Origin is the place where this config file was loaded from
	Origin string `json:"-"`

	// Path to the file where the SSH config to access the workspace is stored
	SSHConfigPath string `json:"sshConfigPath,omitempty"`
}

type WorkspaceIDEConfig struct {
	// Name is the name of the IDE
	Name string `json:"name,omitempty"`

	// Options are the local options that override the global ones
	Options map[string]config.OptionValue `json:"options,omitempty"`
}

type WorkspaceMachineConfig struct {
	// ID is the machine ID to use for this workspace
	ID string `json:"machineId,omitempty"`

	// AutoDelete specifies if the machine should get destroyed when
	// the workspace is destroyed
	AutoDelete bool `json:"autoDelete,omitempty"`
}

type WorkspaceProviderConfig struct {
	// Name is the provider name
	Name string `json:"name,omitempty"`

	// Options are the local options that override the global ones
	Options map[string]config.OptionValue `json:"options,omitempty"`
}

type WorkspaceSource struct {
	// GitRepository is the repository to clone
	GitRepository string `json:"gitRepository,omitempty"`

	// GitBranch is the branch to use
	GitBranch string `json:"gitBranch,omitempty"`

	// GitCommit is the commit SHA to checkout
	GitCommit string `json:"gitCommit,omitempty"`

	// GitPRReference is the pull request reference to checkout
	GitPRReference string `json:"gitPRReference,omitempty"`

	// GitSubPath is the subpath in the repo to use
	GitSubPath string `json:"gitSubDir,omitempty"`

	// LocalFolder is the local folder to use
	LocalFolder string `json:"localFolder,omitempty"`

	// Image is the docker image to use
	Image string `json:"image,omitempty"`

	// Container is the container to use
	Container string `json:"container,omitempty"`
}

type ContainerWorkspaceInfo struct {
	// IDE holds the ide config options
	IDE WorkspaceIDEConfig `json:"ide,omitempty"`

	// CLIOptions holds the cli options
	CLIOptions CLIOptions `json:"cliOptions,omitempty"`

	// Dockerless holds custom dockerless configuration
	Dockerless ProviderDockerlessOptions `json:"dockerless,omitempty"`

	// ContainerTimeout is the timeout in minutes to wait until the agent tries
	// to delete the container.
	ContainerTimeout string `json:"containerInactivityTimeout,omitempty"`

	// Source is a WorkspaceSource to be used inside the container
	Source WorkspaceSource `json:"source,omitempty"`

	// ContentFolder holds the folder where the content is stored
	ContentFolder string `json:"contentFolder,omitempty"`

	// PullFromInsideContainer determines if project should be pulled from Source when container starts
	PullFromInsideContainer types.StrBool `json:"pullFromInsideContainer,omitempty"`

	// Agent holds the agent info
	Agent ProviderAgentConfig `json:"agent,omitempty"`
}

type AgentWorkspaceInfo struct {
	// WorkspaceOrigin is the path where this workspace config originated from
	WorkspaceOrigin string `json:"workspaceOrigin,omitempty"`

	// Workspace holds the workspace info
	Workspace *Workspace `json:"workspace,omitempty"`

	// LastDevContainerConfig can be used as a fallback if the workspace was already started
	// and we lost track of the devcontainer.json
	LastDevContainerConfig *devcontainerconfig.DevContainerConfigWithPath `json:"lastDevContainerConfig,omitempty"`

	// Machine holds the machine info
	Machine *Machine `json:"machine,omitempty"`

	// Agent holds the agent info
	Agent ProviderAgentConfig `json:"agent,omitempty"`

	// CLIOptions holds the cli options
	CLIOptions CLIOptions `json:"cliOptions,omitempty"`

	// Options holds the filled provider options for this workspace
	Options map[string]config.OptionValue `json:"options,omitempty"`

	// ContentFolder holds the folder where the content is stored
	ContentFolder string `json:"contentFolder,omitempty"`

	// Origin holds the folder where this config was loaded from
	Origin string `json:"-"`

	// InjectTimeout specifies how long to wait for the agent to be injected into the dev container
	InjectTimeout time.Duration `json:"injectTimeout,omitempty"`

	// RegistryCache defines the registry to use for caching builds
	RegistryCache string `json:"registryCache,omitempty"`
}

type CLIOptions struct {
	// up options
	flags.GitCredentialsFlags `json:",inline"`
	ID                        string            `json:"id,omitempty"`
	Source                    string            `json:"source,omitempty"`
	IDE                       string            `json:"ide,omitempty"`
	IDEOptions                []string          `json:"ideOptions,omitempty"`
	PrebuildRepositories      []string          `json:"prebuildRepositories,omitempty"`
	DevContainerImage         string            `json:"devContainerImage,omitempty"`
	DevContainerPath          string            `json:"devContainerPath,omitempty"`
	DevContainerSource        string            `json:"devContainerSource,omitempty"`
	EnvironmentTemplate       string            `json:"environmentTemplate,omitempty"`
	WorkspaceEnv              []string          `json:"workspaceEnv,omitempty"`
	WorkspaceEnvFile          []string          `json:"workspaceEnvFile,omitempty"`
	InitEnv                   []string          `json:"initEnv,omitempty"`
	Recreate                  bool              `json:"recreate,omitempty"`
	Reset                     bool              `json:"reset,omitempty"`
	Proxy                     bool              `json:"proxy,omitempty"`
	DisableDaemon             bool              `json:"disableDaemon,omitempty"`
	DaemonInterval            string            `json:"daemonInterval,omitempty"`
	ForceCredentials          bool              `json:"forceCredentials,omitempty"`
	GitCloneStrategy          git.CloneStrategy `json:"gitCloneStrategy,omitempty"`
	FallbackImage             string            `json:"fallbackImage,omitempty"`
	GitSSHSigningKey          string            `json:"gitSshSigningKey,omitempty"`
	ExtraDevContainerPaths    []string          `json:"extraDevContainerPaths,omitempty"`

	// build options
	Repository string   `json:"repository,omitempty"`
	SkipPush   bool     `json:"skipPush,omitempty"`
	Platform   []string `json:"platform,omitempty"`

	ForceBuild            bool   `json:"forceBuild,omitempty"`
	ForceDockerless       bool   `json:"forceDockerless,omitempty"`
	ForceInternalBuildKit bool   `json:"forceInternalBuildKit,omitempty"`
	SSHKey                string `json:"sshkey,omitempty"`
}

type BuildOptions struct {
	CLIOptions

	Platform      string
	RegistryCache string
	ExportCache   bool
	NoBuild       bool
}

func (w WorkspaceSource) String() string {
	if w.GitRepository != "" {
		if w.GitPRReference != "" {
			return WorkspaceSourceGit + w.GitRepository + "@" + w.GitPRReference
		} else if w.GitBranch != "" {
			return WorkspaceSourceGit + w.GitRepository + "@" + w.GitBranch
		} else if w.GitCommit != "" {
			return WorkspaceSourceGit + w.GitRepository + git.CommitDelimiter + w.GitCommit
		}

		return WorkspaceSourceGit + w.GitRepository
	} else if w.LocalFolder != "" {
		return WorkspaceSourceLocal + w.LocalFolder
	} else if w.Image != "" {
		return WorkspaceSourceImage + w.Image
	} else if w.Container != "" {
		return WorkspaceSourceContainer + w.Container
	}

	return ""
}

func ParseWorkspaceSource(source string) *WorkspaceSource {
	if strings.HasPrefix(source, WorkspaceSourceGit) {
		gitRepo, gitPRReference, gitBranch, gitCommit, gitSubdir := git.NormalizeRepository(strings.TrimPrefix(source, WorkspaceSourceGit))
		return &WorkspaceSource{
			GitRepository:  gitRepo,
			GitPRReference: gitPRReference,
			GitBranch:      gitBranch,
			GitCommit:      gitCommit,
			GitSubPath:     gitSubdir,
		}
	} else if strings.HasPrefix(source, WorkspaceSourceLocal) {
		return &WorkspaceSource{
			LocalFolder: strings.TrimPrefix(source, WorkspaceSourceLocal),
		}
	} else if strings.HasPrefix(source, WorkspaceSourceImage) {
		return &WorkspaceSource{
			Image: strings.TrimPrefix(source, WorkspaceSourceImage),
		}
	} else if strings.HasPrefix(source, WorkspaceSourceContainer) {
		return &WorkspaceSource{
			Container: strings.TrimPrefix(source, WorkspaceSourceContainer),
		}
	}

	return nil
}
