package driver

import (
	"context"
	"io"

	"github.com/loft-sh/devpod/pkg/compose"
	config2 "github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/feature"
)

type Driver interface {
	// FindDevContainer returns a running devcontainer details
	FindDevContainer(ctx context.Context, labels []string) (*config.ContainerDetails, error)

	// CommandDevContainer runs the given command inside the devcontainer
	CommandDevContainer(ctx context.Context, containerId, user, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error

	// RunDevContainer runs a devcontainer
	RunDevContainer(
		ctx context.Context,
		parsedConfig *config.DevContainerConfig,
		mergedConfig *config.MergedDevContainerConfig,
		imageName,
		workspaceMount string,
		labels []string,
		ide string,
		ideOptions map[string]config2.OptionValue,
		imageDetails *config.ImageDetails,
	) error

	// DeleteDevContainer deletes the devcontainer
	DeleteDevContainer(ctx context.Context, containerId string, deleteVolumes bool) error

	// StartDevContainer starts the devcontainer
	StartDevContainer(ctx context.Context, containerId string, labels []string) error

	// StopDevContainer stops the devcontainer
	StopDevContainer(ctx context.Context, containerId string) error

	// InspectImage inspects the given image name
	InspectImage(ctx context.Context, imageName string) (*config.ImageDetails, error)

	// BuildDevContainer builds a devcontainer
	BuildDevContainer(
		ctx context.Context,
		labels []string,
		parsedConfig *config.SubstitutedConfig,
		extendedBuildInfo *feature.ExtendedBuildInfo,
		dockerfilePath,
		dockerfileContent string,
		localWorkspaceFolder string,
		options config.BuildOptions,
	) (*config.BuildInfo, error)

	// PushDevContainer pushes the given image to a registry
	PushDevContainer(ctx context.Context, image string) error

	// ComposeHelper returns the compose helper
	ComposeHelper() (*compose.ComposeHelper, error)
}
