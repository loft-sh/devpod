package client

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/loft-sh/devpod/pkg/provider"
)

type BaseClient interface {
	// Provider returns the name of the provider
	Provider() string

	// Context returns the context of the provider
	Context() string

	// RefreshOptions updates the options
	RefreshOptions(ctx context.Context, userOptions []string, reconfigure bool) error

	// Status retrieves the workspace status
	Status(ctx context.Context, options StatusOptions) (Status, error)

	// Stop stops the workspace
	Stop(ctx context.Context, options StopOptions) error

	// Delete deletes the workspace
	Delete(ctx context.Context, options DeleteOptions) error
}

type Client interface {
	BaseClient

	// AgentLocal returns if the agent runs locally
	AgentLocal() bool

	// AgentPath returns the agent path
	AgentPath() string

	// AgentURL returns the agent download url
	AgentURL() string

	// Create creates a new workspace
	Create(ctx context.Context, options CreateOptions) error

	// Start starts the workspace
	Start(ctx context.Context, options StartOptions) error

	// Command creates an SSH tunnel into the workspace
	Command(ctx context.Context, options CommandOptions) error
}

type ProxyClient interface {
	BaseWorkspaceClient

	// Up creates a new remote workspace
	Up(ctx context.Context, options UpOptions) error

	// Ssh starts an ssh tunnel to the workspace container
	Ssh(ctx context.Context, options SshOptions) error
}

type MachineClient interface {
	Client

	// Machine returns the machine of this client
	Machine() string

	// MachineConfig returns the machine config
	MachineConfig() *provider.Machine
}

type BaseWorkspaceClient interface {
	BaseClient

	// Workspace returns the workspace of this provider
	Workspace() string

	// WorkspaceConfig returns the workspace config
	WorkspaceConfig() *provider.Workspace

	// Lock locks the workspace. This is a file lock, which means
	// the workspace is locked across processes.
	Lock(ctx context.Context) error

	// Unlock unlocks the workspace.
	Unlock()
}

type WorkspaceClient interface {
	BaseWorkspaceClient
	Client

	// AgentInjectGitCredentials returns if the credentials helper should get injected
	AgentInjectGitCredentials() bool

	// AgentInjectDockerCredentials returns if the credentials helper should get injected
	AgentInjectDockerCredentials() bool

	// AgentInfo returns the info to send to the agent
	AgentInfo(options provider.CLIOptions) (string, *provider.AgentWorkspaceInfo, error)
}

type InitOptions struct{}

type ValidateOptions struct{}

type StartOptions struct{}

type StopOptions struct{}

type DeleteOptions struct {
	IgnoreNotFound bool   `json:"ignoreNotFound,omitempty"`
	Force          bool   `json:"force,omitempty"`
	GracePeriod    string `json:"gracePeriod,omitempty"`
}

type CreateOptions struct{}

type StatusOptions struct {
	ContainerStatus bool `json:"containerStatus,omitempty"`
}

type CommandOptions struct {
	Command string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

type UpOptions struct {
	provider.CLIOptions

	Debug bool

	Stdin  io.Reader
	Stdout io.Writer
}

type SshOptions struct {
	User string

	Stdin  io.Reader
	Stdout io.Writer
}

type ImportWorkspaceOptions map[string]string

type Status string

const (
	StatusRunning  = "Running"
	StatusBusy     = "Busy"
	StatusStopped  = "Stopped"
	StatusNotFound = "NotFound"
)

func ParseStatus(in string) (Status, error) {
	in = strings.ToUpper(strings.TrimSpace(in))
	switch in {
	case "RUNNING":
		return StatusRunning, nil
	case "BUSY":
		return StatusBusy, nil
	case "STOPPED":
		return StatusStopped, nil
	case "NOTFOUND":
		return StatusNotFound, nil
	default:
		return StatusNotFound, fmt.Errorf("error parsing status: '%s' unrecognized status, needs to be one of: %s", in, []string{StatusRunning, StatusBusy, StatusStopped, StatusNotFound})
	}
}

type WorkspaceStatus struct {
	ID       string `json:"id,omitempty"`
	Context  string `json:"context,omitempty"`
	Provider string `json:"provider,omitempty"`
	State    string `json:"state,omitempty"`
}
