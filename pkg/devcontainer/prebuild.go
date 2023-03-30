package devcontainer

import (
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/hash"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/pkg/errors"
	"runtime"
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

func calculatePrebuildHash(parsedConfig *config.DevContainerConfig, dockerfileContent string, log log.Logger) (string, error) {
	// TODO: is it a good idea to delete customizations before calculating the hash?
	parsedConfig = config.CloneDevContainerConfig(parsedConfig)
	parsedConfig.Customizations = nil

	// marshal the config
	configStr, err := json.Marshal(parsedConfig)
	if err != nil {
		return "", err
	}

	log.Debugf("Prebuild hash from: %s %s %s", runtime.GOARCH, string(configStr), dockerfileContent)
	return "devpod-" + hash.String(runtime.GOARCH + string(configStr) + dockerfileContent)[:32], nil
}
