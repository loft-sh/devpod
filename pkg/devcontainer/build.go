package devcontainer

import (
	"fmt"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/feature"
	"github.com/loft-sh/devpod/pkg/devcontainer/metadata"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/devpod/pkg/dockerfile"
	"github.com/loft-sh/devpod/pkg/hash"
	"github.com/loft-sh/devpod/pkg/id"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	ImageDetails  *docker.ImageDetails
	ImageMetadata *config.ImageMetadataConfig
	ImageName     string
}

func (r *Runner) build(parsedConfig *config.SubstitutedConfig) (*BuildInfo, error) {
	if isDockerFileConfig(parsedConfig.Config) {
		return r.buildAndExtendImage(parsedConfig)
	}

	return r.extendImage(parsedConfig)
}

func (r *Runner) extendImage(parsedConfig *config.SubstitutedConfig) (*BuildInfo, error) {
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
			ImageMetadata: imageBuildInfo.Metadata,
			ImageName:     imageBase,
		}, nil
	}

	// build the image
	return r.buildImage(parsedConfig, extendedBuildInfo, "", "")
}

func (r *Runner) buildAndExtendImage(parsedConfig *config.SubstitutedConfig) (*BuildInfo, error) {
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
	return r.buildImage(parsedConfig, extendedBuildInfo, dockerFilePath, string(dockerFileContent))
}

func (r *Runner) buildImage(parsedConfig *config.SubstitutedConfig, extendedBuildInfo *feature.ExtendedBuildInfo, dockerfilePath, dockerfileContent string) (*BuildInfo, error) {
	imageName := r.getImageName()

	// extra args?
	finalDockerfilePath := dockerfilePath
	finalDockerfileContent := string(dockerfileContent)
	extraArgs := []string{}
	if extendedBuildInfo != nil && extendedBuildInfo.FeaturesBuildInfo != nil {
		featureBuildInfo := extendedBuildInfo.FeaturesBuildInfo

		// cleanup features folder after we are done building
		if featureBuildInfo.FeaturesFolder != "" {
			defer os.RemoveAll(featureBuildInfo.FeaturesFolder)
		}

		// rewrite dockerfile
		finalDockerfileContent = dockerfile.RemoveSyntaxVersion(string(dockerfileContent))
		finalDockerfileContent = strings.Join([]string{
			featureBuildInfo.DockerfilePrefixContent,
			strings.TrimSpace(finalDockerfileContent),
			featureBuildInfo.DockerfileContent,
		}, "\n")

		// write dockerfile with features
		finalDockerfilePath = filepath.Join(featureBuildInfo.FeaturesFolder, "Dockerfile-with-features")
		err := os.WriteFile(finalDockerfilePath, []byte(finalDockerfileContent), 0666)
		if err != nil {
			return nil, errors.Wrap(err, "write Dockerfile with features")
		}

		// track additional build args to include below
		for k, v := range featureBuildInfo.BuildKitContexts {
			extraArgs = append(extraArgs, "--build-context", k+"="+v)
		}
		for k, v := range featureBuildInfo.BuildArgs {
			extraArgs = append(extraArgs, "--build-arg", k+"="+v)
		}
	}

	// add label
	if extendedBuildInfo != nil && extendedBuildInfo.MetadataLabel != "" {
		extraArgs = append(extraArgs, "--label", extendedBuildInfo.MetadataLabel)
	}

	// build args
	args := []string{
		"buildx",
		"build",
		//"--build-arg", "BUILDKIT_INLINE_CACHE=1",
		"--load",
		"-f", finalDockerfilePath,
		"-t", imageName,
	}

	// extra args
	args = append(args, extraArgs...)

	// target stage
	if extendedBuildInfo != nil && extendedBuildInfo.FeaturesBuildInfo != nil && extendedBuildInfo.FeaturesBuildInfo.OverrideTarget != "" {
		args = append(args, "--target", extendedBuildInfo.FeaturesBuildInfo.OverrideTarget)
	} else if parsedConfig.Config.Build.Target != "" {
		args = append(args, "--target", parsedConfig.Config.Build.Target)
	}

	// cache
	for _, cacheFrom := range parsedConfig.Config.Build.CacheFrom {
		args = append(args, "--cache-from", cacheFrom)
	}

	// build args
	for k, v := range parsedConfig.Config.Build.Args {
		args = append(args, "--build-arg", k+"="+v)
	}

	// context
	args = append(args, getContextPath(parsedConfig.Config))
	r.Log.Debugf("Running docker command: docker %s", strings.Join(args, " "))
	writer := r.Log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	err := r.Docker.Run(args, nil, writer, writer)
	if err != nil {
		return nil, errors.Wrap(err, "build image")
	}

	imageDetails, err := r.Docker.InspectImage(imageName, false)
	if err != nil {
		return nil, errors.Wrap(err, "get image details")
	}

	return &BuildInfo{
		ImageDetails:  imageDetails,
		ImageMetadata: extendedBuildInfo.MetadataConfig,
		ImageName:     imageName,
	}, nil
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
	ImageDetails *docker.ImageDetails
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
