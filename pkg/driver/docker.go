package driver

import (
	"context"

	"github.com/loft-sh/devpod/pkg/compose"
	config2 "github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/feature"
)

type DockerDriver interface {
	Driver

	// InspectImage inspects the given image name
	InspectImage(ctx context.Context, imageName string) (*config.ImageDetails, error)

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
		options config.BuildOptions,
	) (*config.BuildInfo, error)

	// PushDevContainer pushes the given image to a registry
	PushDevContainer(ctx context.Context, image string) error

	// ComposeHelper returns the compose helper
	ComposeHelper() (*compose.ComposeHelper, error)
}
