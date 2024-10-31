package driver

import (
	"context"

	"github.com/loft-sh/devpod/pkg/compose"
	config2 "github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/feature"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/devpod/pkg/provider"
)

type DockerDriver interface {
	Driver

	// InspectImage inspects the given image name
	InspectImage(ctx context.Context, imageName string) (*config.ImageDetails, error)

	// GetImageTag returns latest tag for input image id
	GetImageTag(ctx context.Context, imageName string) (string, error)

	// RunDockerDevContainer runs a docker devcontainer
	RunDockerDevContainer(
		ctx context.Context,
		workspaceId string,
		options *RunOptions,
		parsedConfig *config.DevContainerConfig,
		init *bool,
		ide string,
		ideOptions map[string]config2.OptionValue,
	) error

	// BuildDevContainer builds a devcontainer
	BuildDevContainer(
		ctx context.Context,
		prebuildHash string,
		parsedConfig *config.SubstitutedConfig,
		extendedBuildInfo *feature.ExtendedBuildInfo,
		dockerfilePath,
		dockerfileContent string,
		localWorkspaceFolder string,
		options provider.BuildOptions,
	) (*config.BuildInfo, error)

	// PushDevContainer pushes the given image to a registry
	PushDevContainer(ctx context.Context, image string) error

	TagDevContainer(ctx context.Context, image, tag string) error

	// ComposeHelper returns the compose helper
	ComposeHelper() (*compose.ComposeHelper, error)

	// DockerHellper returns the docker helper
	DockerHelper() (*docker.DockerHelper, error)
}
