package client

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/provider"
)

type Client interface {
	// Provider returns the name of the provider
	Provider() string

	// Context returns the context of the provider
	Context() string

	// RefreshOptions updates the options
	RefreshOptions(ctx context.Context, userOptions []string) error

	// AgentLocal returns if the agent runs locally
	AgentLocal() bool

	// AgentPath returns the agent path
	AgentPath() string

	// AgentURL returns the agent download url
	AgentURL() string

	// Create creates a new workspace
	Create(ctx context.Context, options CreateOptions) error

	// Delete deletes the workspace
	Delete(ctx context.Context, options DeleteOptions) error

	// Start starts the workspace
	Start(ctx context.Context, options StartOptions) error

	// Stop stops the workspace
	Stop(ctx context.Context, options StopOptions) error

	// Status retrieves the workspace status
	Status(ctx context.Context, options StatusOptions) (Status, error)

	// Command creates an SSH tunnel into the workspace
	Command(ctx context.Context, options CommandOptions) error
}

type MachineClient interface {
	Client

	// Machine returns the machine of this client
	Machine() string

	// MachineConfig returns the machine config
	MachineConfig() *provider.Machine
}

type WorkspaceClient interface {
	Client

	// Workspace returns the workspace of this provider
	Workspace() string

	// WorkspaceConfig returns the workspace config
	WorkspaceConfig() *provider.Workspace

	// AgentConfig returns the agent config to send to the agent
	AgentConfig() provider.ProviderAgentConfig

	// AgentInfo returns the info to send to the agent
	AgentInfo() (string, *provider.AgentWorkspaceInfo, error)
}

type InitOptions struct{}

type ValidateOptions struct{}

type StartOptions struct{}

type StopOptions struct{}

type DeleteOptions struct {
	Force       bool
	GracePeriod *time.Duration
}

type CreateOptions struct{}

type StatusOptions struct {
	ContainerStatus bool
}

type CommandOptions struct {
	Command string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

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
