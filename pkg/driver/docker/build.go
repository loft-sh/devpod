package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/loft-sh/devpod/pkg/devcontainer/build"
	"github.com/loft-sh/devpod/pkg/devcontainer/buildkit"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/feature"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (d *dockerDriver) BuildDevContainer(
	ctx context.Context,
	prebuildHash string,
	parsedConfig *config.SubstitutedConfig,
	extendedBuildInfo *feature.ExtendedBuildInfo,
	dockerfilePath,
	dockerfileContent string,
	localWorkspaceFolder string,
	options provider.BuildOptions,
) (*config.BuildInfo, error) {
	// check if image build is necessary
	imageName := build.GetImageName(localWorkspaceFolder, prebuildHash)
	if options.Repository == "" && !options.ForceBuild {
		imageDetails, err := d.Docker.InspectImage(ctx, imageName, false)
		if err == nil && imageDetails != nil {
			// local image found
			d.Log.Infof("Found existing local image %s", imageName)
			return &config.BuildInfo{
				ImageDetails:  imageDetails,
				ImageMetadata: extendedBuildInfo.MetadataConfig,
				ImageName:     imageName,
				PrebuildHash:  prebuildHash,
				RegistryCache: options.RegistryCache,
				Tags:          options.Tag,
			}, nil
		} else if err != nil {
			d.Log.Debugf("Error trying to find local image %s: %v", imageName, err)
		}
	}

	// check if we shouldn't build
	if options.NoBuild {
		return nil, fmt.Errorf("you cannot build in this mode. Please run 'devpod up' to rebuild the container")
	}

	// get build options
	buildOptions, err := build.NewOptions(dockerfilePath, dockerfileContent, parsedConfig, extendedBuildInfo, imageName, options, prebuildHash)
	if err != nil {
		return nil, err
	}
	d.Log.Debug("Using registry cache", options.RegistryCache)

	// build image
	writer := d.Log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// check if docker buildx exists
	if options.Platform != "" {
		d.Log.Infof("Build for platform '%s'...", options.Platform)
	}

	builder := d.Docker.Builder
	if (builder == docker.DockerBuilderDefault || builder == docker.
		DockerBuilderBuildX) && d.buildxExists(ctx) && !options.ForceInternalBuildKit {
		builder = docker.DockerBuilderBuildX
	} else {
		builder = docker.DockerBuilderBuildKit
	}

	switch builder {
	case docker.DockerBuilderBuildX:
		if d.buildxExists(ctx) {
			d.Log.Info("Build with docker buildx...")
			err := d.buildxBuild(ctx, writer, options.Platform, buildOptions)
			if err != nil {
				return nil, errors.Wrap(err, "buildx build")
			}
		} else {
			return nil, fmt.Errorf("buildx is not available on your host. Use buildkit builder")
		}
	case docker.DockerBuilderBuildKit:
		d.Log.Info("Build with internal buildkit...")
		err := d.internalBuild(ctx, writer, options.Platform, buildOptions)
		if err != nil {
			return nil, errors.Wrap(err, "internal build")
		}
	case docker.DockerBuilderDefault:
		return nil, fmt.Errorf("invalid docker builder: %s", builder)
	}

	// inspect image
	imageDetails, err := d.Docker.InspectImage(ctx, imageName, false)
	if err != nil {
		return nil, errors.Wrap(err, "get image details")
	}

	return &config.BuildInfo{
		ImageDetails:  imageDetails,
		ImageMetadata: extendedBuildInfo.MetadataConfig,
		ImageName:     imageName,
		PrebuildHash:  prebuildHash,
		RegistryCache: options.RegistryCache,
		Tags:          options.Tag,
	}, nil
}

func (d *dockerDriver) buildxExists(ctx context.Context) bool {
	buf := &bytes.Buffer{}
	err := d.Docker.Run(ctx, []string{"buildx", "version"}, nil, buf, buf)

	return (err == nil) || d.Docker.IsPodman()
}

func (d *dockerDriver) internalBuild(ctx context.Context, writer io.Writer, platform string, options *build.BuildOptions) error {
	dockerClient, err := docker.NewClient(ctx, d.Log)
	if err != nil {
		return errors.Wrap(err, "create docker client")
	}
	defer dockerClient.Close()

	buildKitClient, err := buildkit.NewDockerClient(ctx, dockerClient)
	if err != nil {
		return errors.Wrap(err, "create buildkit client")
	}
	defer buildKitClient.Close()

	err = buildkit.Build(ctx, buildKitClient, writer, platform, options, d.Log)
	if err != nil {
		return errors.Wrap(err, "build")
	}

	return nil
}

func (d *dockerDriver) buildxBuild(ctx context.Context, writer io.Writer, platform string, options *build.BuildOptions) error {
	// build args
	args := []string{
		"buildx",
		"build",
		"-f", options.Dockerfile,
	}

	// add load
	if options.Load {
		args = append(args, "--load")
	}

	// docker images
	for _, image := range options.Images {
		args = append(args, "-t", image)
	}

	// build args
	for k, v := range options.BuildArgs {
		args = append(args, "--build-arg", k+"="+v)
	}

	// build contexts
	for k, v := range options.Contexts {
		args = append(args, "--build-context", k+"="+v)
	}

	// target stage
	if options.Target != "" {
		args = append(args, "--target", options.Target)
	}

	// platform
	if platform != "" {
		args = append(args, "--platform", platform)
	}

	// cache
	for _, cacheFrom := range options.CacheFrom {
		args = append(args, "--cache-from", cacheFrom)
	}
	for _, cacheTo := range options.CacheTo {
		args = append(args, "--cache-to", cacheTo)
	}

	// add additional build cli options
	args = append(args, options.CliOpts...)

	// context
	args = append(args, options.Context)

	// run command
	d.Log.Debugf("Running docker %s: docker %s", d.Docker.DockerCommand, strings.Join(args, " "))
	err := d.Docker.Run(ctx, args, nil, writer, writer)
	if err != nil {
		return errors.Wrap(err, "build image")
	}

	return nil
}
