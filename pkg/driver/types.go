package driver

import (
	"context"
	"github.com/loft-sh/devpod/pkg/compose"
	config2 "github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/feature"
	"io"
)

type Driver interface {
	// CommandDevContainer runs the given command inside the devcontainer
	CommandDevContainer(ctx context.Context, id, user, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error

	// DeleteDevContainer deletes the devcontainer
	DeleteDevContainer(id string) error

	// FindDevContainer returns a running devcontainer details
	FindDevContainer(labels []string) (*config.ContainerDetails, error)

	// InspectImage inspects the given image name
	InspectImage(imageName string) (*config.ImageDetails, error)

	// RunDevContainer runs a devcontainer
	RunDevContainer(
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
	PushDevContainer(image string) error

	// BuildDevContainer builds a devcontainer
	BuildDevContainer(
		parsedConfig *config.SubstitutedConfig,
		extendedBuildInfo *feature.ExtendedBuildInfo,
		dockerfilePath,
		dockerfileContent string,
		imageName string,
		prebuildHash string,
		options config.BuildOptions,
	) (*config.BuildInfo, error)

	// StartDevContainer starts the devcontainer
	StartDevContainer(id string, labels []string) error

	// StopDevContainer stops the devcontainer
	StopDevContainer(id string) error

	// ComposeHelper returns the compose helper
	ComposeHelper() (*compose.ComposeHelper, error)
}
