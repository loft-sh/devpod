package devcontainer

import (
	"fmt"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/pkg/errors"
)

func (r *Runner) Build(options config.BuildOptions) (string, error) {
	substitutedConfig, _, err := r.prepare()
	if err != nil {
		return "", err
	}

	// check if we need to build container
	buildInfo, err := r.build(substitutedConfig, config.BuildOptions{ForceRebuild: options.ForceRebuild, PushRepository: options.PushRepository})
	if err != nil {
		return "", errors.Wrap(err, "build image")
	}

	// prebuild already exists
	prebuildImage := options.PushRepository + ":" + buildInfo.PrebuildHash
	if !options.ForceRebuild && buildInfo.ImageName == prebuildImage {
		return buildInfo.ImageName, nil
	}

	// check if we can push image
	err = image.CheckPushPermissions(prebuildImage)
	if err != nil {
		return "", fmt.Errorf("cannot push to repository %s. Please make sure you are logged into the registry and credentials are available. (Error: %v)", prebuildImage, err)
	}

	// push the image to the registry
	err = r.Driver.PushDevContainer(prebuildImage)
	if err != nil {
		return "", errors.Wrap(err, "push image")
	}

	return prebuildImage, nil
}
