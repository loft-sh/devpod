package client

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/loft-sh/api/v4/pkg/devpod"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/provider"
	"golang.org/x/crypto/ssh"
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

// ProxyClient executes it's commands on the platform
type ProxyClient interface {
	BaseWorkspaceClient

	// Create creates a new remote workspace
	Create(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error

	// Up creates a new remote workspace
	Up(ctx context.Context, options UpOptions) error

	// Ssh starts an ssh tunnel to the workspace container
	Ssh(ctx context.Context, options SshOptions) error
}

// DaemonClient connects to workspaces through a shared daemon
type DaemonClient interface {
	BaseWorkspaceClient

	// Create creates a new remote workspace
	Create(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error

	// Up start a new remote workspace
	Up(ctx context.Context, options UpOptions) (*config.Result, error)

	// SSHClients returns an SSH client for the tool and one for the actual user
	SSHClients(ctx context.Context, user string) (*ssh.Client, *ssh.Client, error)

	// CheckWorkspaceReachable checks if the given workspace is reachable from the current machine
	CheckWorkspaceReachable(ctx context.Context) error

	// DirectTunnel forwards stdio to the workspace
	DirectTunnel(ctx context.Context, stdin io.Reader, stdout io.Writer) error

	// Ping tries to ping a workspace and prints results to stdout
	Ping(ctx context.Context, stdout io.Writer) error
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
	AgentInjectGitCredentials(options provider.CLIOptions) bool

	// AgentInjectDockerCredentials returns if the credentials helper should get injected
	AgentInjectDockerCredentials(options provider.CLIOptions) bool

	// AgentInfo returns the info to send to the agent
	AgentInfo(options provider.CLIOptions) (string, *provider.AgentWorkspaceInfo, error)
}

type InitOptions struct{}

type ValidateOptions struct{}

type StartOptions struct{}

type StopOptions struct {
	Platform devpod.PlatformOptions `json:"platform,omitempty"`
}

type DeleteOptions struct {
	Platform devpod.PlatformOptions `json:"platform,omitempty"`

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

type User struct {
	Name string `json:"name,omitempty"`
	UID  string `json:"uid,omitempty"`
}
