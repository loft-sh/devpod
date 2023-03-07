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

	// Type defines the type of the provider. Defaults to Server
	Type ProviderType `json:"type,omitempty"`

	// Description is the provider description
	Description string `json:"description,omitempty"`

	// Options are the provider options.
	Options map[string]*ProviderOption `json:"options,omitempty"`

	// Agent allows you to override agent configuration
	Agent ProviderAgentConfig `json:"agent,omitempty"`

	// Exec holds the provider commands
	Exec ProviderCommands `json:"exec,omitempty"`

	// Binaries is an optional field to specify a binary to execute the commands
	Binaries map[string][]*ProviderBinary `json:"binaries,omitempty"`
}

type ProviderAgentConfig struct {
	// Path is the path inside the server devpod will expect the agent
	Path string `json:"path,omitempty"`

	// DownloadURL is the base url where to download the agent from
	DownloadURL string `json:"downloadURL,omitempty"`

	// Timeout is the timeout in minutes to wait until the agent tries
	// to turn of the server. Defaults to 1 hour.
	Timeout string `json:"inactivityTimeout,omitempty"`

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
}

type ProviderAgentConfigExec struct {
	// Shutdown is the remote command to run when the remote machine
	// should shutdown.
	Shutdown types.StrArray `json:"shutdown,omitempty"`
}

type ProviderType string

const (
	ProviderTypeMachine = "Machine"
	ProviderTypeDirect  = "Direct"
)

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
	// Init is run directly after `devpod use provider`
	Init types.StrArray `json:"init,omitempty"`

	// Validate is run directly after init and after the variables have been resolved.
	Validate types.StrArray `json:"validate,omitempty"`

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

	// ValidationPattern is a regex pattern to validate the value
	ValidationPattern string `json:"validationPattern,omitempty"`

	// ValidationMessage is the message that appears if the user enters an invalid option
	ValidationMessage string `json:"validationMessage,omitempty"`

	// Allowed values for this option.
	Enum []string `json:"enum,omitempty"`

	// Hidden specifies if the option should be hidden
	Hidden bool `json:"hidden,omitempty"`

	// Local will never send the option to the server
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
	if c.Type == ProviderTypeDirect {
		return false
	}
	if len(c.Exec.Create) > 0 {
		return true
	}
	return false
}
