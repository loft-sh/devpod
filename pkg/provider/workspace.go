package provider

import (
	"strings"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/git"
	"github.com/loft-sh/devpod/pkg/types"
)

var (
	WorkspaceSourceGit   = "git:"
	WorkspaceSourceLocal = "local:"
	WorkspaceSourceImage = "image:"
)

type Workspace struct {
	// ID is the workspace id to use
	ID string `json:"id,omitempty"`

	// UID is used to identify this specific workspace
	UID string `json:"uid,omitempty"`

	// Folder is the local folder where workspace related contents will be stored
	Folder string `json:"folder,omitempty"`

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

	// CreationTimestamp is the timestamp when this workspace was created
	CreationTimestamp types.Time `json:"creationTimestamp,omitempty"`

	// LastUsedTimestamp holds the timestamp when this workspace was last accessed
	LastUsedTimestamp types.Time `json:"lastUsed,omitempty"`

	// Context is the context where this config file was loaded from
	Context string `json:"context,omitempty"`

	// Origin is the place where this config file was loaded from
	Origin string `json:"-"`
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

	// UID is the machine UID to use for this workspace
	UID string `json:"machineUid,omitempty"`

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

	// LocalFolder is the local folder to use
	LocalFolder string `json:"localFolder,omitempty"`

	// Image is the docker image to use
	Image string `json:"image,omitempty"`
}

type AgentWorkspaceInfo struct {
	// Workspace holds the workspace info
	Workspace *Workspace `json:"workspace,omitempty"`

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
}

type CLIOptions struct {
	// up options
	ID                   string   `json:"id,omitempty"`
	Source               string   `json:"source,omitempty"`
	IDE                  string   `json:"ide,omitempty"`
	IDEOptions           []string `json:"ideOptions,omitempty"`
	PrebuildRepositories []string `json:"prebuildRepositories,omitempty"`
	DevContainerImage    string   `json:"devContainerImage,omitempty"`
	DevContainerPath     string   `json:"devContainerPath,omitempty"`
	WorkspaceEnv         []string `json:"workspaceEnv,omitempty"`
	Recreate             bool     `json:"recreate,omitempty"`
	Proxy                bool     `json:"proxy,omitempty"`
	DisableDaemon        bool     `json:"disableDaemon,omitempty"`
	DaemonInterval       string   `json:"daemonInterval,omitempty"`

	// build options
	Repository string   `json:"repository,omitempty"`
	SkipPush   bool     `json:"skipPush,omitempty"`
	Platform   []string `json:"platform,omitempty"`

	// TESTING
	ForceBuild            bool `json:"forceBuild,omitempty"`
	ForceDockerless       bool `json:"forceDockerless,omitempty"`
	ForceInternalBuildKit bool `json:"forceInternalBuildKit,omitempty"`
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
	}

	return ""
}

func ParseWorkspaceSource(source string) *WorkspaceSource {
	if strings.HasPrefix(source, WorkspaceSourceGit) {
		gitRepo, gitPRReference, gitBranch, gitCommit := git.NormalizeRepository(strings.TrimPrefix(source, WorkspaceSourceGit))
		return &WorkspaceSource{
			GitRepository:  gitRepo,
			GitPRReference: gitPRReference,
			GitBranch:      gitBranch,
			GitCommit:      gitCommit,
		}
	} else if strings.HasPrefix(source, WorkspaceSourceLocal) {
		return &WorkspaceSource{
			LocalFolder: strings.TrimPrefix(source, WorkspaceSourceLocal),
		}
	} else if strings.HasPrefix(source, WorkspaceSourceImage) {
		return &WorkspaceSource{
			Image: strings.TrimPrefix(source, WorkspaceSourceImage),
		}
	}

	return nil
}
