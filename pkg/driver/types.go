package driver

import (
	"context"
	"io"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
)

// Driver is the default interface for DevPod drivers
type Driver interface {
	// FindDevContainer returns a running devcontainer details
	FindDevContainer(ctx context.Context, workspaceID string) (*config.ContainerDetails, error)

	// CommandDevContainer runs the given command inside the devcontainer
	CommandDevContainer(ctx context.Context, workspaceID, user, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error

	// RunDevContainer runs a devcontainer
	RunDevContainer(ctx context.Context, workspaceID string, options *RunOptions) error

	// TargetArchitecture returns the architecture of the container runtime. e.g. amd64 or arm64
	TargetArchitecture(ctx context.Context, workspaceID string) (string, error)

	// DeleteDevContainer deletes the devcontainer
	DeleteDevContainer(ctx context.Context, workspaceID string) error

	// StartDevContainer starts the devcontainer
	StartDevContainer(ctx context.Context, workspaceID string) error

	// StopDevContainer stops the devcontainer
	StopDevContainer(ctx context.Context, workspaceID string) error

	// GetContainerLogs returns the logs of the devcontainer
	GetDevContainerLogs(ctx context.Context, workspaceID string, stdout io.Writer, stderr io.Writer) error
}

type ReprovisioningDriver interface {
	Driver

	// CanReprovision returns true if the driver can reprovision the devcontainer
	CanReprovision() bool
}

// RunOptions are the options for running a container
type RunOptions struct {
	// UID is a unique identifier for this workspace
	UID string `json:"uid,omitempty"`

	// Image is the image to run
	Image string `json:"image,omitempty"`

	// User is the user to run the container as
	User string `json:"user,omitempty"`

	// Entrypoint is the entrypoint of the container
	Entrypoint string `json:"entrypoint,omitempty"`

	// Cmd are the cmd for the entrypoint
	Cmd []string `json:"cmd,omitempty"`

	// Env are additional environment variables to set
	Env map[string]string `json:"env,omitempty"`

	// CapAdd are additional capabilities for the container
	CapAdd []string `json:"capAdd,omitempty"`

	// SecurityOpt are additional security options
	SecurityOpt []string `json:"securityOpt,omitempty"`

	// Labels are labels to set on the container
	Labels []string `json:"labels,omitempty"`

	// Privileged indicates if the container should run with elevated permissions
	Privileged *bool `json:"privileged,omitempty"`

	// WorkspaceMount is the mount where the workspace should get mounted
	WorkspaceMount *config.Mount `json:"workspaceMount,omitempty"`

	// Mounts are additional mounts on the container. Supported are volume and bind mounts.
	// Bind mounts are expected to get copied from local to remote once. Volume mounts are expected
	// to be persisted for the lifetime of the container.
	Mounts []*config.Mount `json:"mounts,omitempty"`
}
