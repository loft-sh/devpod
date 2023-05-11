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
	// CommandDevContainer runs the given command inside the devcontainer
	CommandDevContainer(ctx context.Context, id, user, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error

	// DeleteDevContainer deletes the devcontainer
	DeleteDevContainer(ctx context.Context, id string, deleteVolumes bool) error

	// FindDevContainer returns a running devcontainer details
	FindDevContainer(ctx context.Context, labels []string) (*config.ContainerDetails, error)

	// StartDevContainer starts the devcontainer
	StartDevContainer(ctx context.Context, id string, labels []string) error

	// StopDevContainer stops the devcontainer
	StopDevContainer(ctx context.Context, id string) error

	// InspectImage inspects the given image name
	InspectImage(ctx context.Context, imageName string) (*config.ImageDetails, error)

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

	// PushDevContainer pushes the given image to a registry
	PushDevContainer(ctx context.Context, image string) error

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

	// ComposeHelper returns the compose helper
	ComposeHelper() (*compose.ComposeHelper, error)
}
