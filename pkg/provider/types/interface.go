package types

import (
	"context"
	"github.com/loft-sh/devpod/pkg/config"
	"io"
)

type Provider interface {
	// Name returns the name of the provider
	Name() string
}

type WorkspaceTunnelOptions struct {
	Token  string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type WorkspaceCreateOptions struct{}

type WorkspaceDestroyOptions struct{}

type WorkspaceStatusOptions struct{}

type WorkspaceStartOptions struct{}

type WorkspaceStopOptions struct{}

type WorkspaceProvider interface {
	Provider

	// WorkspaceCreate creates a new workspace
	WorkspaceCreate(ctx context.Context, workspace *config.Workspace, options WorkspaceCreateOptions) error

	// WorkspaceDestroy destroys the given workspace
	WorkspaceDestroy(ctx context.Context, workspace *config.Workspace, options WorkspaceDestroyOptions) error

	// WorkspaceStart starts a stopped workspace
	WorkspaceStart(ctx context.Context, workspace *config.Workspace, options WorkspaceStartOptions) error

	// WorkspaceStop stops a given workspace
	WorkspaceStop(ctx context.Context, workspace *config.Workspace, options WorkspaceStopOptions) error

	// WorkspaceStatus retrieves the workspace status for the given workspace
	WorkspaceStatus(ctx context.Context, workspace *config.Workspace, options WorkspaceStatusOptions) (Status, error)

	// WorkspaceTunnel creates an SSH tunnel into the workspace
	WorkspaceTunnel(ctx context.Context, workspace *config.Workspace, options WorkspaceTunnelOptions) error
}

type StartOptions struct{}

type StopOptions struct{}

type DestroyOptions struct{}

type CreateOptions struct{}

type StatusOptions struct{}

type RunCommandOptions struct {
	Command string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

type ServerProvider interface {
	Provider

	// Create creates a server for the given workspace
	Create(ctx context.Context, workspace *config.Workspace, options CreateOptions) error

	// Destroy destroys the given workspace server
	Destroy(ctx context.Context, workspace *config.Workspace, options DestroyOptions) error

	// Start starts a previously stopped workspace
	Start(ctx context.Context, workspace *config.Workspace, options StartOptions) error

	// Stop stops a running workspace
	Stop(ctx context.Context, workspace *config.Workspace, options StopOptions) error

	// Status retrieves the server status for the given workspace
	Status(ctx context.Context, workspace *config.Workspace, options StatusOptions) (Status, error)

	// RunCommand runs a given command on the server
	RunCommand(ctx context.Context, workspace *config.Workspace, options RunCommandOptions) error
}

type Status string

const (
	StatusRunning  = "Running"
	StatusBusy     = "Busy"
	StatusStopped  = "Stopped"
	StatusNotFound = "NotFound"
)
