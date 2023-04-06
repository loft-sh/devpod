package docker

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/devcontainer/build"
	"github.com/loft-sh/devpod/pkg/devcontainer/buildkit"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/feature"
	"github.com/loft-sh/devpod/pkg/devcontainer/metadata"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/devpod/pkg/dockerfile"
	"github.com/loft-sh/devpod/pkg/hash"
	"github.com/loft-sh/devpod/pkg/id"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

func (d *dockerDriver) BuildDevContainer(
	ctx context.Context,
	labels []string,
	parsedConfig *config.SubstitutedConfig,
	extendedBuildInfo *feature.ExtendedBuildInfo,
	dockerfilePath,
	dockerfileContent string,
	localWorkspaceFolder string,
	options config.BuildOptions,
) (*config.BuildInfo, error) {
	prebuildHash, err := config.CalculatePrebuildHash(parsedConfig.Config, runtime.GOARCH, dockerfileContent, d.Log)
	if err != nil {
		return nil, err
	}

	// check if there is a prebuild image
	if !options.ForceRebuild {
		devPodCustomizations := config.GetDevPodCustomizations(parsedConfig.Config)
		if options.PushRepository != "" {
			options.PrebuildRepositories = append(options.PrebuildRepositories, options.PushRepository)
		}
		options.PrebuildRepositories = append(options.PrebuildRepositories, devPodCustomizations.PrebuildRepository...)
		d.Log.Debugf("Try to find prebuild image %s in repositories %s", prebuildHash, strings.Join(options.PrebuildRepositories, ","))
		for _, prebuildRepo := range options.PrebuildRepositories {
			prebuildImage := prebuildRepo + ":" + prebuildHash
			img, err := image.GetImage(prebuildImage)
			if err == nil && img != nil {
				// prebuild image found
				d.Log.Infof("Found existing prebuilt image %s", prebuildImage)

				// inspect image
				imageDetails, err := d.InspectImage(ctx, prebuildImage)
				if err != nil {
					return nil, errors.Wrap(err, "get image details")
				}

				return &config.BuildInfo{
					ImageDetails:  imageDetails,
					ImageMetadata: extendedBuildInfo.MetadataConfig,
					ImageName:     prebuildImage,
					PrebuildHash:  prebuildHash,
				}, nil
			} else if err != nil {
				d.Log.Debugf("Error trying to find prebuild image %s: %v", prebuildImage, err)
			}
		}
	}

	// check if we shouldn't build
	if options.NoBuild {
		return nil, fmt.Errorf("you cannot build in this mode. Please run 'devpod up' to rebuild the container")
	}

	// build the image
	imageName := getImageName(localWorkspaceFolder)

	// get build options
	buildOptions, err := CreateBuildOptions(dockerfilePath, dockerfileContent, parsedConfig, extendedBuildInfo, imageName, options.PushRepository, prebuildHash)
	if err != nil {
		return nil, err
	}

	// build image
	writer := d.Log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// check if docker buildx exists
	if d.buildxExists(ctx) {
		d.Log.Infof("Build with docker buildx")
		err := d.buildxBuild(ctx, writer, buildOptions)
		if err != nil {
			return nil, errors.Wrap(err, "buildx build")
		}
	} else {
		d.Log.Infof("Build with internal buildkit")
		err := d.internalBuild(ctx, writer, buildOptions)
		if err != nil {
			return nil, errors.Wrap(err, "internal build")
		}
	}

	// inspect image
	imageDetails, err := d.Docker.InspectImage(imageName, false)
	if err != nil {
		return nil, errors.Wrap(err, "get image details")
	}

	return &config.BuildInfo{
		ImageDetails:  imageDetails,
		ImageMetadata: extendedBuildInfo.MetadataConfig,
		ImageName:     imageName,
		PrebuildHash:  prebuildHash,
	}, nil
}

func CreateBuildOptions(
	dockerfilePath, dockerfileContent string,
	parsedConfig *config.SubstitutedConfig,
	extendedBuildInfo *feature.ExtendedBuildInfo,
	imageName string,
	pushRepository string,
	prebuildHash string,
) (*build.BuildOptions, error) {
	// extra args?
	finalDockerfilePath := dockerfilePath
	finalDockerfileContent := string(dockerfileContent)
	buildOptions := &build.BuildOptions{
		BuildArgs: parsedConfig.Config.Build.Args,
		Labels:    map[string]string{},
		Contexts:  map[string]string{},
		Load:      true,
	}
	if buildOptions.BuildArgs == nil {
		buildOptions.BuildArgs = map[string]string{}
	}

	// get extended build info
	if extendedBuildInfo != nil && extendedBuildInfo.FeaturesBuildInfo != nil {
		featureBuildInfo := extendedBuildInfo.FeaturesBuildInfo

		// cleanup features folder after we are done building
		if featureBuildInfo.FeaturesFolder != "" {
			defer os.RemoveAll(featureBuildInfo.FeaturesFolder)
		}

		// rewrite dockerfile
		finalDockerfileContent = dockerfile.RemoveSyntaxVersion(string(dockerfileContent))
		finalDockerfileContent = strings.TrimSpace(strings.Join([]string{
			featureBuildInfo.DockerfilePrefixContent,
			strings.TrimSpace(finalDockerfileContent),
			featureBuildInfo.DockerfileContent,
		}, "\n"))

		// write dockerfile with features
		finalDockerfilePath = filepath.Join(featureBuildInfo.FeaturesFolder, "Dockerfile-with-features")
		err := os.WriteFile(finalDockerfilePath, []byte(finalDockerfileContent), 0666)
		if err != nil {
			return nil, errors.Wrap(err, "write Dockerfile with features")
		}

		// track additional build args to include below
		for k, v := range featureBuildInfo.BuildKitContexts {
			buildOptions.Contexts[k] = v
		}
		for k, v := range featureBuildInfo.BuildArgs {
			buildOptions.BuildArgs[k] = v
		}
	}

	// add label
	if extendedBuildInfo != nil && extendedBuildInfo.MetadataLabel != "" {
		buildOptions.Labels[metadata.ImageMetadataLabel] = extendedBuildInfo.MetadataLabel
	}

	// target
	if extendedBuildInfo != nil && extendedBuildInfo.FeaturesBuildInfo != nil && extendedBuildInfo.FeaturesBuildInfo.OverrideTarget != "" {
		buildOptions.Target = extendedBuildInfo.FeaturesBuildInfo.OverrideTarget
	} else if parsedConfig.Config.Build.Target != "" {
		buildOptions.Target = parsedConfig.Config.Build.Target
	}

	// other options
	buildOptions.Dockerfile = finalDockerfilePath
	if imageName != "" {
		buildOptions.Images = append(buildOptions.Images, imageName)
	}
	if pushRepository != "" {
		buildOptions.Images = append(buildOptions.Images, pushRepository+":"+prebuildHash)
	}
	buildOptions.Context = getContextPath(parsedConfig.Config)

	// add build arg
	if buildOptions.BuildArgs == nil {
		buildOptions.BuildArgs = map[string]string{}
	}
	buildOptions.BuildArgs["BUILDKIT_INLINE_CACHE"] = "1"
	return buildOptions, nil
}

func getImageName(localWorkspaceFolder string) string {
	imageHash := hash.String(localWorkspaceFolder)[:5]
	return "vsc-" + id.ToDockerImageName(filepath.Base(localWorkspaceFolder)) + "-" + imageHash
}

func (d *dockerDriver) buildxExists(ctx context.Context) bool {
	buf := &bytes.Buffer{}
	err := d.Docker.Run(ctx, []string{"buildx", "version"}, nil, buf, buf)
	if err != nil {
		return false
	}

	return true
}

func (d *dockerDriver) internalBuild(ctx context.Context, writer io.Writer, options *build.BuildOptions) error {
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

	err = buildkit.Build(ctx, buildKitClient, writer, options, d.Log)
	if err != nil {
		return errors.Wrap(err, "build")
	}

	return nil
}

func (d *dockerDriver) buildxBuild(ctx context.Context, writer io.Writer, options *build.BuildOptions) error {
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

	// cache
	for _, cacheFrom := range options.CacheFrom {
		args = append(args, "--cache-from", cacheFrom)
	}

	// context
	args = append(args, options.Context)

	// run command
	d.Log.Debugf("Running docker command: docker %s", strings.Join(args, " "))
	err := d.Docker.Run(ctx, args, nil, writer, writer)
	if err != nil {
		return errors.Wrap(err, "build image")
	}

	return nil
}

func getContextPath(parsedConfig *config.DevContainerConfig) string {
	context := ""
	dockerfilePath := ""
	if parsedConfig.Dockerfile != "" {
		context = parsedConfig.Context
		dockerfilePath = parsedConfig.Dockerfile
	} else if parsedConfig.Build.Dockerfile != "" {
		context = parsedConfig.Build.Context
		dockerfilePath = parsedConfig.Build.Dockerfile
	}

	configDir := path.Dir(filepath.ToSlash(parsedConfig.Origin))
	if context != "" {
		return filepath.FromSlash(path.Join(configDir, context))
	} else if dockerfilePath != "" {
		return filepath.FromSlash(path.Join(configDir, path.Dir(dockerfilePath)))
	}

	return configDir
}
