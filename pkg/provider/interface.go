package provider

import (
	"context"
	"fmt"
	"io"
	"strings"
)

type Provider interface {
	// Name returns the name of the provider
	Name() string

	// Description returns the description of the provider
	Description() string

	// Options returns the available options of the provider
	Options() map[string]*ProviderOption

	// AgentConfig returns the agent config
	AgentConfig() (*ProviderAgentConfig, error)

	// Init initializes the provider with new config options
	Init(ctx context.Context, workspace *Workspace, options InitOptions) error

	// Validate validates that the provider and all other values are correctly setup
	Validate(ctx context.Context, workspace *Workspace, options ValidateOptions) error
}

type InitOptions struct{}

type ValidateOptions struct{}

type WorkspaceTunnelOptions struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type WorkspaceCreateOptions struct{}

type WorkspaceDeleteOptions struct {
	Force bool
}

type WorkspaceStatusOptions struct{}

type WorkspaceStartOptions struct{}

type WorkspaceStopOptions struct{}

type WorkspaceProvider interface {
	Provider

	// Create creates a new workspace
	Create(ctx context.Context, workspace *Workspace, options WorkspaceCreateOptions) error

	// Delete deletes the given workspace
	Delete(ctx context.Context, workspace *Workspace, options WorkspaceDeleteOptions) error

	// Start starts a stopped workspace
	Start(ctx context.Context, workspace *Workspace, options WorkspaceStartOptions) error

	// Stop stops a given workspace
	Stop(ctx context.Context, workspace *Workspace, options WorkspaceStopOptions) error

	// Status retrieves the workspace status for the given workspace
	Status(ctx context.Context, workspace *Workspace, options WorkspaceStatusOptions) (Status, error)

	// Tunnel creates an SSH tunnel into the workspace
	Tunnel(ctx context.Context, workspace *Workspace, options WorkspaceTunnelOptions) error
}

type StartOptions struct{}

type StopOptions struct{}

type DeleteOptions struct {
	Force bool
}

type CreateOptions struct{}

type StatusOptions struct{}

type CommandOptions struct {
	Command string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

type ServerProvider interface {
	Provider

	// Create creates a server for the given workspace
	Create(ctx context.Context, workspace *Workspace, options CreateOptions) error

	// Delete deletes the given workspace server
	Delete(ctx context.Context, workspace *Workspace, options DeleteOptions) error

	// Start starts a previously stopped workspace
	Start(ctx context.Context, workspace *Workspace, options StartOptions) error

	// Stop stops a running workspace
	Stop(ctx context.Context, workspace *Workspace, options StopOptions) error

	// Status retrieves the server status for the given workspace
	Status(ctx context.Context, workspace *Workspace, options StatusOptions) (Status, error)

	// Command runs a given command on the server
	Command(ctx context.Context, workspace *Workspace, options CommandOptions) error
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
