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

	// Options are the provider options
	Options map[string]*ProviderOption `json:"options,omitempty"`

	// Exec holds the provider commands
	Exec ProviderCommands `json:"exec,omitempty"`

	// Binaries is an optional field to specify a binary to execute the commands
	Binaries []*ProviderBinary `json:"binaries,omitempty"`
}

type ProviderType string

const (
	ProviderTypeServer    = "Server"
	ProviderTypeWorkspace = "Workspace"
)

type ProviderBinary struct {
	// The current OS
	OS string `json:"os"`

	// The current Arch
	Arch string `json:"arch"`

	// The binary url to download from or relative path to use
	Path string `json:"path"`
}

type ProviderCommands struct {
	// Init is run directly after `devpod use provider`
	Init types.StrArray `json:"init,omitempty"`

	// Command executes a command on the server
	Command types.StrArray `json:"command,omitempty"`

	// Tunnel creates a tunnel to the workspace
	Tunnel types.StrArray `json:"tunnel,omitempty"`

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
	// Default value if the user omits this option from their configuration.
	Default string `json:"default,omitempty"`

	// A description of the option displayed to the user by a supporting tool.
	Description string `json:"description,omitempty"`

	// ValidationPattern is a regex pattern to validate the value
	ValidationPattern string `json:"validationPattern,omitempty"`

	// ValidationMessage is the message that appears if the user enters an invalid option
	ValidationMessage string `json:"validationMessage,omitempty"`

	// Allowed values for this option.
	Enum []string `json:"enum,omitempty"`
}
