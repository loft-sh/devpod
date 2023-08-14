package devcontainer

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/pkg/errors"
)

func (r *Runner) Build(ctx context.Context, options config.BuildOptions) (string, error) {
	substitutedConfig, _, err := r.prepare(options.CLIOptions)
	if err != nil {
		return "", err
	}

	if options.Repository == "" && len(config.GetDevPodCustomizations(substitutedConfig.Config).PrebuildRepository) == 0 {
		return "", fmt.Errorf("repository needs to be specified")
	}

	// check if we need to build container
	buildInfo, err := r.build(ctx, substitutedConfig, options)
	if err != nil {
		return "", errors.Wrap(err, "build image")
	}

	// prebuild already exists
	var prebuildImage string
	if options.Repository != "" {
		prebuildImage = options.Repository + ":" + buildInfo.PrebuildHash
	} else if config.GetDevPodCustomizations(substitutedConfig.Config).PrebuildRepository[0] != "" {
		prebuildImage = config.GetDevPodCustomizations(substitutedConfig.Config).PrebuildRepository[0] + ":" + buildInfo.PrebuildHash
	}

	if buildInfo.ImageName == prebuildImage {
		return buildInfo.ImageName, nil
	}

	// should we push?
	if options.SkipPush {
		return prebuildImage, nil
	}

	// check if we can push image
	err = image.CheckPushPermissions(prebuildImage)
	if err != nil {
		return "", fmt.Errorf(
			"cannot push to repository %s. Please make sure you are logged into the registry and credentials are available. (Error: %w)",
			prebuildImage,
			err,
		)
	}

	// push the image to the registry
	err = r.Driver.PushDevContainer(context.TODO(), prebuildImage)
	if err != nil {
		return "", errors.Wrap(err, "push image")
	}

	return prebuildImage, nil
}
