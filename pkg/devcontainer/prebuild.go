package devcontainer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver"
	"github.com/loft-sh/devpod/pkg/driver/docker"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
)

func (r *runner) Build(ctx context.Context, options provider.BuildOptions) (string, error) {
	dockerDriver, ok := r.Driver.(driver.DockerDriver)
	if !ok {
		return "", fmt.Errorf("building only supported with docker driver")
	}

	substitutedConfig, substitutionContext, err := r.getSubstitutedConfig(options.CLIOptions)
	if err != nil {
		return "", err
	}

	prebuildRepo := getPrebuildRepository(substitutedConfig)

	if !options.SkipPush && options.Repository == "" && prebuildRepo == "" {
		return "", fmt.Errorf("repository needs to be specified")
	}

	// remove build information
	defer func() {
		contextPath := config.GetContextPath(substitutedConfig.Config)
		_ = os.RemoveAll(filepath.Join(contextPath, config.DevPodContextFeatureFolder))
	}()

	// check if we need to build container
	buildInfo, err := r.build(ctx, substitutedConfig, substitutionContext, options)
	if err != nil {
		return "", errors.Wrap(err, "build image")
	}

	// have a fallback value for PrebuildHash
	// in some cases it may be empty, and this would lead to
	// invalid reference format during image pushing.
	if buildInfo.PrebuildHash == "" {
		buildInfo.PrebuildHash = "latest"
	}

	// prebuild already exists
	var prebuildImage string
	if options.Repository != "" {
		prebuildImage = options.Repository + ":" + buildInfo.PrebuildHash
	} else if prebuildRepo != "" {
		prebuildImage = prebuildRepo + ":" + buildInfo.PrebuildHash
	} else {
		prebuildImage = docker.GetImageName(r.LocalWorkspaceFolder, buildInfo.PrebuildHash)
	}

	if buildInfo.ImageName == prebuildImage {
		return buildInfo.ImageName, nil
	}

	// should we push?
	if options.SkipPush {
		return prebuildImage, nil
	}

	if isDockerComposeConfig(substitutedConfig.Config) {
		r.Log.Debug("Tagging image prebuild=%s buildInfo=%s", prebuildImage, buildInfo.ImageName)
		err = dockerDriver.TagDevContainer(ctx, buildInfo.ImageName, prebuildImage)
		if err != nil {
			return "", errors.Wrap(err, "tag image")
		}
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
	err = dockerDriver.PushDevContainer(ctx, prebuildImage)
	if err != nil {
		return "", errors.Wrap(err, "push image")
	}

	return prebuildImage, nil
}

func getPrebuildRepository(substitutedConfig *config.SubstitutedConfig) string {
	if len(config.GetDevPodCustomizations(substitutedConfig.Config).PrebuildRepository) > 0 {
		return config.GetDevPodCustomizations(substitutedConfig.Config).PrebuildRepository[0]
	}

	return ""
}
