package devcontainer

import (
	"fmt"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/feature"
	"github.com/loft-sh/devpod/pkg/devcontainer/metadata"
	"github.com/loft-sh/devpod/pkg/dockerfile"
	"github.com/loft-sh/devpod/pkg/hash"
	"github.com/loft-sh/devpod/pkg/id"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"strings"
)

func (r *Runner) build(parsedConfig *config.SubstitutedConfig, options config.BuildOptions) (*config.BuildInfo, error) {
	if isDockerFileConfig(parsedConfig.Config) {
		return r.buildAndExtendImage(parsedConfig, options)
	}

	return r.extendImage(parsedConfig, options)
}

func (r *Runner) extendImage(parsedConfig *config.SubstitutedConfig, options config.BuildOptions) (*config.BuildInfo, error) {
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
		return &config.BuildInfo{
			ImageDetails:  imageBuildInfo.ImageDetails,
			ImageMetadata: extendedBuildInfo.MetadataConfig,
			ImageName:     imageBase,
		}, nil
	}

	// build the image
	return r.buildImage(parsedConfig, extendedBuildInfo, "", "", options)
}

func (r *Runner) buildAndExtendImage(parsedConfig *config.SubstitutedConfig, options config.BuildOptions) (*config.BuildInfo, error) {
	dockerFilePath, err := r.getDockerfilePath(parsedConfig.Config)
	if err != nil {
		return nil, err
	}

	dockerFileContent, err := os.ReadFile(dockerFilePath)
	if err != nil {
		return nil, err
	}

	// ensure there is a target to choose for us
	imageBase := config.DockerfileDefaultTarget
	if parsedConfig.Config.Build.Target != "" {
		imageBase = parsedConfig.Config.Build.Target
	} else {
		lastTargetName, modifiedDockerfileContents, err := dockerfile.EnsureDockerfileHasFinalStageName(string(dockerFileContent), config.DockerfileDefaultTarget)
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

func (r *Runner) getImageBuildInfoFromImage(imageName string) (*config.ImageBuildInfo, error) {
	imageDetails, err := r.Driver.InspectImage(imageName)
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

	return &config.ImageBuildInfo{
		ImageDetails: imageDetails,
		User:         user,
		Metadata:     imageMetadata,
	}, nil
}

func (r *Runner) getImageBuildInfoFromDockerfile(dockerFileContent string, buildArgs map[string]string, target string) (*config.ImageBuildInfo, error) {
	parsedDockerfile, err := dockerfile.Parse(dockerFileContent)
	if err != nil {
		return nil, errors.Wrap(err, "parse dockerfile")
	}

	baseImage := parsedDockerfile.FindBaseImage(buildArgs, target)
	imageDetails, err := r.Driver.InspectImage(baseImage)
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

	return &config.ImageBuildInfo{
		Dockerfile: parsedDockerfile,
		User:       user,
		Metadata:   imageMetadataConfig,
	}, nil
}

func (r *Runner) buildImage(parsedConfig *config.SubstitutedConfig, extendedBuildInfo *feature.ExtendedBuildInfo, dockerfilePath, dockerfileContent string, options config.BuildOptions) (*config.BuildInfo, error) {
	prebuildHash, err := calculatePrebuildHash(parsedConfig.Config, dockerfileContent, r.Log)
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
		r.Log.Debugf("Try to find prebuild image %s in repositories %s", prebuildHash, strings.Join(options.PrebuildRepositories, ","))
		for _, prebuildRepo := range options.PrebuildRepositories {
			prebuildImage := prebuildRepo + ":" + prebuildHash
			img, err := image.GetImage(prebuildImage)
			if err == nil && img != nil {
				// prebuild image found
				r.Log.Infof("Found existing prebuilt image %s", prebuildImage)

				// inspect image
				imageDetails, err := r.Driver.InspectImage(prebuildImage)
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
				r.Log.Debugf("Error trying to find prebuild image %s: %v", prebuildImage, err)
			}
		}
	}

	// check if we shouldn't build
	if options.NoBuild {
		return nil, fmt.Errorf("you cannot build in this mode. Please run 'devpod up' to rebuild the container")
	}

	// build the image
	imageName := r.getImageName()
	return r.Driver.BuildDevContainer(parsedConfig, extendedBuildInfo, dockerfilePath, dockerfileContent, imageName, prebuildHash, options)
}

func (r *Runner) getImageName() string {
	imageHash := hash.String(r.LocalWorkspaceFolder)[:5]
	return "vsc-" + id.ToDockerImageName(filepath.Base(r.LocalWorkspaceFolder)) + "-" + imageHash
}
