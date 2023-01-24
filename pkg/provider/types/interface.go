package types

import (
	"context"
	"github.com/loft-sh/devpod/pkg/id"
	"io"
)

func NewWorkspace(repository string) *Workspace {
	return &Workspace{
		ID:         id.WorkspaceID(repository),
		Repository: repository,
	}
}

func NewWorkspaceWithID(repository, id string) *Workspace {
	return &Workspace{
		ID:         id,
		Repository: repository,
	}
}

type Workspace struct {
	// ID is the workspace id to use
	ID string
	// Repository is the repository to clone
	Repository string
}

type ApplyOptions struct {
	// DisableSnapshot ignores an existing snapshot
	DisableSnapshot bool
}

type DestroyOptions struct {
	// Force should delete the workspace even though terraform has failed
	// to complete.
	Force bool
}

type ApplySnapshotOptions struct{}

type DestroySnapshotOptions struct{}

type StopOptions struct{}

type DialSSHOptions struct{}

type RemoteCommandOptions struct{}

type Provider interface {
	// Apply applies the workspace.
	Apply(ctx context.Context, workspace *Workspace, options ApplyOptions) error

	// Destroy cleans up an existing workspace.
	Destroy(ctx context.Context, workspace *Workspace, options DestroyOptions) error

	// ApplySnapshot creates a snapshot of the workspace
	ApplySnapshot(ctx context.Context, workspace *Workspace, options ApplySnapshotOptions) error

	// DestroySnapshot destroys the snapshot of the workspace
	DestroySnapshot(ctx context.Context, workspace *Workspace, options DestroySnapshotOptions) error

	// Stop stops a workspace.
	Stop(ctx context.Context, workspace *Workspace, options StopOptions) error

	// RemoteCommandHost returns a handler to run remote commands on the container host
	RemoteCommandHost(ctx context.Context, workspace *Workspace, options RemoteCommandOptions) (RemoteCommandHandler, error)
}

type RemoteCommandHandler interface {
	// Run executes a remote command
	Run(ctx context.Context, cmd string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error

	// Close closes the command handler
	Close() error
}
