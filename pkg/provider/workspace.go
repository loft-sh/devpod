package provider

import (
	"github.com/loft-sh/devpod/pkg/types"
)

type Workspace struct {
	// ID is the workspace id to use
	ID string `json:"id,omitempty"`

	// Folder is the local folder where workspace related contents will be stored
	Folder string `json:"folder,omitempty"`

	// Provider is the provider used to create this workspace
	Provider WorkspaceProviderConfig `json:"provider,omitempty"`

	// Source is the source where this workspace will be created from
	Source WorkspaceSource `json:"source,omitempty"`

	// CreationTimestamp is the timestamp when this workspace was created
	CreationTimestamp types.Time `json:"creationTimestamp,omitempty"`

	// Context is the context where this config file was loaded from
	Context string `json:"context,omitempty"`

	// Origin is the place where this config file was loaded from
	Origin string `json:"-"`
}

type WorkspaceProviderConfig struct {
	// Name is the provider name
	Name string `json:"name,omitempty"`

	// Options are the provider options used to create the workspace
	Options map[string]OptionValue `json:"options,omitempty"`
}

type OptionValue struct {
	// Value is the value of the option
	Value string `json:"value,omitempty"`

	// Expires is the time when this value will expire
	Expires *types.Time `json:"retrieved,omitempty"`
}

type WorkspaceSource struct {
	// GitRepository is the repository to clone
	GitRepository string `json:"gitRepository,omitempty"`

	// GitBranch is the branch to use
	GitBranch string `json:"gitBranch,omitempty"`

	// GitCommit is the commit to use
	GitCommit string `json:"gitCommit,omitempty"`

	// LocalFolder is the local folder to use
	LocalFolder string `json:"localFolder,omitempty"`

	// Image is the docker image to use
	Image string `json:"image,omitempty"`
}

func (w WorkspaceSource) String() string {
	if w.GitRepository != "" {
		if w.GitBranch != "" {
			return w.GitRepository + "@" + w.GitBranch
		}
		if w.GitCommit != "" {
			return w.GitRepository + "@" + w.GitCommit
		}

		return w.GitRepository
	}

	if w.LocalFolder != "" {
		return w.LocalFolder
	}

	return w.Image
}
