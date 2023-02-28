package devcontainer

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
	"strings"
)

const (
	DockerIDLabel           = "dev.containers.id"
	DockerfileDefaultTarget = "dev_container_auto_added_stage_label"
)

type BuildInfo struct {
	ImageDetails  *config.ImageDetails
	ImageMetadata *config.ImageMetadataConfig
	ImageName     string
	PrebuildHash  string
}

type BuildOptions struct {
	ForceRebuild bool

	PrebuildRepositories []string
	PushRepository       string
}

func (r *Runner) build(parsedConfig *config.SubstitutedConfig, options BuildOptions) (*BuildInfo, error) {
	if isDockerFileConfig(parsedConfig.Config) {
		return r.buildAndExtendImage(parsedConfig, options)
	}

	return r.extendImage(parsedConfig, options)
}

func (r *Runner) extendImage(parsedConfig *config.SubstitutedConfig, options BuildOptions) (*BuildInfo, error) {
	imageBase := parsedConfig.Config.Image
	imageBuildInfo, err := r.getImageBuildInfoFromImage(imageBase)
	if err != nil {
		return nil, errors.Wrap(err, "get image build info")
	}

	// get extend image build info
	extendedBuildInfo, err := feature.GetExtendedBuildInfo(r.SubstitutionContext, imageBuildInfo.Metadata, imageBuildInfo.User, imageBase, parsedConfig, r.Log)
	if err != nil {
		return nil, errors.Wrap(err, "get extended build info")
	}

	// no need to build here
	if extendedBuildInfo == nil || extendedBuildInfo.FeaturesBuildInfo == nil {
		return &BuildInfo{
			ImageDetails:  imageBuildInfo.ImageDetails,
			ImageMetadata: extendedBuildInfo.MetadataConfig,
			ImageName:     imageBase,
		}, nil
	}

	// build the image
	return r.buildImage(parsedConfig, extendedBuildInfo, "", "", options)
}

func (r *Runner) buildAndExtendImage(parsedConfig *config.SubstitutedConfig, options BuildOptions) (*BuildInfo, error) {
	dockerFilePath, err := r.getDockerfilePath(parsedConfig.Config)
	if err != nil {
		return nil, err
	}

	dockerFileContent, err := os.ReadFile(dockerFilePath)
	if err != nil {
		return nil, err
	}

	// ensure there is a target to choose for us
	imageBase := DockerfileDefaultTarget
	if parsedConfig.Config.Build.Target != "" {
		imageBase = parsedConfig.Config.Build.Target
	} else {
		lastTargetName, modifiedDockerfileContents, err := dockerfile.EnsureDockerfileHasFinalStageName(string(dockerFileContent), DockerfileDefaultTarget)
		if err != nil {
			return nil, err
		} else if modifiedDockerfileContents != "" {
			dockerFileContent = []byte(modifiedDockerfileContents)
		}

		imageBase = lastTargetName
	}

	// get image build info
	imageBuildInfo, err := r.getImageBuildInfoFromDockerfile(string(dockerFileContent), parsedConfig.Config.Build.Args, parsedConfig.Config.Build.Target)
	if err != nil {
		return nil, errors.Wrap(err, "get image build info")
	}

	// get extend image build info
	extendedBuildInfo, err := feature.GetExtendedBuildInfo(r.SubstitutionContext, imageBuildInfo.Metadata, imageBuildInfo.User, imageBase, parsedConfig, r.Log)
	if err != nil {
		return nil, errors.Wrap(err, "get extended build info")
	}

	// build the image
	return r.buildImage(parsedConfig, extendedBuildInfo, dockerFilePath, string(dockerFileContent), options)
}

func (r *Runner) buildImage(parsedConfig *config.SubstitutedConfig, extendedBuildInfo *feature.ExtendedBuildInfo, dockerfilePath, dockerfileContent string, options BuildOptions) (*BuildInfo, error) {
	prebuildHash, err := calculatePrebuildHash(parsedConfig.Config, dockerfileContent)
	if err != nil {
		return nil, err
	}

	// check if there is a prebuild image
	if !options.ForceRebuild {
		devPodCustomizations := config.GetDevPodCustomizations(parsedConfig.Config)
		options.PrebuildRepositories = append(options.PrebuildRepositories, devPodCustomizations.PrebuildRepo...)
		for _, prebuildRepo := range options.PrebuildRepositories {
			prebuildImage := prebuildRepo + ":" + prebuildHash
			img, err := image.GetImage(prebuildImage)
			if err == nil && img != nil {
				// prebuild image found
				r.Log.Infof("Found existing prebuilt image %s", prebuildImage)

				// inspect image
				imageDetails, err := r.Docker.InspectImage(prebuildImage, true)
				if err != nil {
					return nil, errors.Wrap(err, "get image details")
				}

				return &BuildInfo{
					ImageDetails:  imageDetails,
					ImageMetadata: extendedBuildInfo.MetadataConfig,
					ImageName:     prebuildImage,
					PrebuildHash:  prebuildHash,
				}, nil
			}
		}
	}

	imageName := r.getImageName()

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
	buildOptions.Images = append(buildOptions.Images, imageName)
	if options.PushRepository != "" {
		buildOptions.Images = append(buildOptions.Images, options.PushRepository+":"+prebuildHash)
	}
	buildOptions.Context = getContextPath(parsedConfig.Config)

	// add build arg
	if buildOptions.BuildArgs == nil {
		buildOptions.BuildArgs = map[string]string{}
	}
	buildOptions.BuildArgs["BUILDKIT_INLINE_CACHE"] = "1"

	// build image
	writer := r.Log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// check if docker buildx exists
	if r.buildxExists() {
		r.Log.Infof("Build with docker buildx")
		err := r.buildxBuild(writer, buildOptions)
		if err != nil {
			return nil, errors.Wrap(err, "buildx build")
		}
	} else {
		r.Log.Infof("Build with internal buildkit")
		err := r.internalBuild(context.Background(), writer, buildOptions)
		if err != nil {
			return nil, errors.Wrap(err, "internal build")
		}
	}

	// inspect image
	imageDetails, err := r.Docker.InspectImage(imageName, false)
	if err != nil {
		return nil, errors.Wrap(err, "get image details")
	}

	return &BuildInfo{
		ImageDetails:  imageDetails,
		ImageMetadata: extendedBuildInfo.MetadataConfig,
		ImageName:     imageName,
		PrebuildHash:  prebuildHash,
	}, nil
}

func (r *Runner) buildxExists() bool {
	buf := &bytes.Buffer{}
	err := r.Docker.Run(context.TODO(), []string{"buildx", "version"}, nil, buf, buf)
	if err != nil {
		return false
	}

	return true
}

func (r *Runner) internalBuild(ctx context.Context, writer io.Writer, options *build.BuildOptions) error {
	dockerClient, err := docker.NewClient(ctx, r.Log)
	if err != nil {
		return errors.Wrap(err, "create docker client")
	}
	defer dockerClient.Close()

	buildKitClient, err := buildkit.NewDockerClient(ctx, dockerClient)
	if err != nil {
		return errors.Wrap(err, "create buildkit client")
	}
	defer buildKitClient.Close()

	err = buildkit.Build(ctx, buildKitClient, writer, options)
	if err != nil {
		return errors.Wrap(err, "build")
	}

	return nil
}

func (r *Runner) buildxBuild(writer io.Writer, options *build.BuildOptions) error {
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
	r.Log.Debugf("Running docker command: docker %s", strings.Join(args, " "))
	err := r.Docker.Run(context.TODO(), args, nil, writer, writer)
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

type ImageBuildInfo struct {
	User     string
	Metadata *config.ImageMetadataConfig

	// Either on of these will be filled as will
	Dockerfile   *dockerfile.Dockerfile
	ImageDetails *config.ImageDetails
}

func (r *Runner) getImageBuildInfoFromImage(imageName string) (*ImageBuildInfo, error) {
	imageDetails, err := r.Docker.InspectImage(imageName, true)
	if err != nil {
		return nil, err
	}

	user := "root"
	if imageDetails.Config.User != "" {
		user = imageDetails.Config.User
	}

	imageMetadata, err := metadata.GetImageMetadata(imageDetails, r.SubstitutionContext, r.Log)
	if err != nil {
		return nil, errors.Wrap(err, "get image metadata")
	}

	return &ImageBuildInfo{
		ImageDetails: imageDetails,
		User:         user,
		Metadata:     imageMetadata,
	}, nil
}

func (r *Runner) getImageBuildInfoFromDockerfile(dockerFileContent string, buildArgs map[string]string, target string) (*ImageBuildInfo, error) {
	parsedDockerfile, err := dockerfile.Parse(dockerFileContent)
	if err != nil {
		return nil, errors.Wrap(err, "parse dockerfile")
	}

	baseImage := parsedDockerfile.FindBaseImage(buildArgs, target)
	imageDetails, err := r.Docker.InspectImage(baseImage, true)
	if err != nil {
		return nil, errors.Wrapf(err, "inspect image %s", baseImage)
	}

	// find user
	user := parsedDockerfile.FindUserStatement(buildArgs, config.ListToObject(imageDetails.Config.Env), target)
	if user == "" {
		user = imageDetails.Config.User
	}
	if user == "" {
		user = "root"
	}

	// parse metadata from image details
	imageMetadataConfig, err := metadata.GetImageMetadata(imageDetails, r.SubstitutionContext, r.Log)
	if err != nil {
		return nil, errors.Wrap(err, "get image metadata")
	}

	return &ImageBuildInfo{
		Dockerfile: parsedDockerfile,
		User:       user,
		Metadata:   imageMetadataConfig,
	}, nil
}

func (r *Runner) getImageName() string {
	imageHash := hash.Sha256(r.LocalWorkspaceFolder)[:5]
	return "vsc-" + id.ToDockerImageName(filepath.Base(r.LocalWorkspaceFolder)) + "-" + imageHash
}

func (r *Runner) getDockerfilePath(parsedConfig *config.DevContainerConfig) (string, error) {
	if parsedConfig.Origin == "" {
		return "", fmt.Errorf("couldn't find path where config was loaded from")
	}

	configFileDir := filepath.Dir(parsedConfig.Origin)
	dockerfile := parsedConfig.Dockerfile
	if dockerfile == "" {
		dockerfile = parsedConfig.Build.Dockerfile
	}

	dockerfilePath := filepath.Join(configFileDir, dockerfile)
	_, err := os.Stat(dockerfilePath)
	if err != nil {
		return "", fmt.Errorf("couldn't find Dockerfile at %s", dockerfilePath)
	}

	return dockerfilePath, nil
}
