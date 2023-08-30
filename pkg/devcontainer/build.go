package devcontainer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devpod/pkg/compose"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/feature"
	"github.com/loft-sh/devpod/pkg/devcontainer/metadata"
	"github.com/loft-sh/devpod/pkg/dockerfile"
	"github.com/loft-sh/devpod/pkg/driver"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/pkg/errors"
)

func (r *Runner) build(ctx context.Context, parsedConfig *config.SubstitutedConfig, options config.BuildOptions) (*config.BuildInfo, error) {
	if isDockerFileConfig(parsedConfig.Config) {
		return r.buildAndExtendImage(ctx, parsedConfig, options)
	} else if isDockerComposeConfig(parsedConfig.Config) {
		composeHelper, err := r.composeHelper()
		if err != nil {
			return nil, errors.Wrap(err, "find docker compose")
		}

		envFiles, err := r.getEnvFiles()
		if err != nil {
			return nil, errors.Wrap(err, "get env files")
		}

		composeFiles, err := r.getDockerComposeFilePaths(parsedConfig, envFiles)
		if err != nil {
			return nil, errors.Wrap(err, "get docker compose file paths")
		}

		var composeGlobalArgs []string
		for _, configFile := range composeFiles {
			composeGlobalArgs = append(composeGlobalArgs, "-f", configFile)
		}

		for _, envFile := range envFiles {
			composeGlobalArgs = append(composeGlobalArgs, "--env-file", envFile)
		}

		r.Log.Debugf("Loading docker compose project %+v", composeFiles)
		project, err := compose.LoadDockerComposeProject(composeFiles, envFiles)
		if err != nil {
			return nil, errors.Wrap(err, "load docker compose project")
		}
		project.Name = composeHelper.GetProjectName(r.ID)
		r.Log.Debugf("Loaded project %s", project.Name)

		service := parsedConfig.Config.Service
		composeService, err := project.GetService(service)
		if err != nil {
			return nil, fmt.Errorf("service '%s' configured in devcontainer.json not found in Docker Compose configuration", service)
		}

		originalImageName := composeService.Image
		if originalImageName == "" {
			originalImageName, err = composeHelper.GetDefaultImage(project.Name, service)
			if err != nil {
				return nil, errors.Wrap(err, "get default image")
			}
		}

		overrideBuildImageName, _, imageMetadata, _, err := r.buildAndExtendDockerCompose(ctx, parsedConfig, project, composeHelper, &composeService, composeGlobalArgs)
		if err != nil {
			return nil, errors.Wrap(err, "build and extend docker-compose")
		}

		currentImageName := overrideBuildImageName
		if currentImageName == "" {
			currentImageName = originalImageName
		}

		imageDetails, err := r.inspectImage(ctx, currentImageName)
		if err != nil {
			return nil, errors.Wrap(err, "inspect image")
		}

		return &config.BuildInfo{
			ImageDetails:  imageDetails,
			ImageMetadata: imageMetadata,
			ImageName:     overrideBuildImageName,
			PrebuildHash:  "",
		}, nil
	}

	return r.extendImage(ctx, parsedConfig, options)
}

func (r *Runner) extendImage(ctx context.Context, parsedConfig *config.SubstitutedConfig, options config.BuildOptions) (*config.BuildInfo, error) {
	imageBase := parsedConfig.Config.Image
	imageBuildInfo, err := r.getImageBuildInfoFromImage(ctx, imageBase)
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
	return r.buildImage(ctx, parsedConfig, extendedBuildInfo, "", "", options)
}

func (r *Runner) buildAndExtendImage(ctx context.Context, parsedConfig *config.SubstitutedConfig, options config.BuildOptions) (*config.BuildInfo, error) {
	dockerFilePath, err := r.getDockerfilePath(parsedConfig.Config)
	if err != nil {
		return nil, err
	}

	dockerFileContent, err := os.ReadFile(dockerFilePath)
	if err != nil {
		return nil, err
	}

	// ensure there is a target to choose for us
	var imageBase string
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
	return r.buildImage(ctx, parsedConfig, extendedBuildInfo, dockerFilePath, string(dockerFileContent), options)
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

func (r *Runner) getImageBuildInfoFromImage(ctx context.Context, imageName string) (*config.ImageBuildInfo, error) {
	imageDetails, err := r.inspectImage(ctx, imageName)
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
	if baseImage == "" {
		return nil, fmt.Errorf("find base image %s", target)
	}

	imageDetails, err := r.inspectImage(context.TODO(), baseImage)
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

func (r *Runner) buildImage(ctx context.Context, parsedConfig *config.SubstitutedConfig, extendedBuildInfo *feature.ExtendedBuildInfo, dockerfilePath, dockerfileContent string, options config.BuildOptions) (*config.BuildInfo, error) {
	targetArch, err := r.Driver.TargetArchitecture(ctx, r.ID)
	if err != nil {
		return nil, err
	}

	prebuildHash, err := config.CalculatePrebuildHash(parsedConfig.Config, options.Platform, targetArch, config.GetContextPath(parsedConfig.Config), dockerfilePath, dockerfileContent, r.Log)
	if err != nil {
		return nil, err
	}

	// check if there is a prebuild image
	if !options.ForceBuild {
		devPodCustomizations := config.GetDevPodCustomizations(parsedConfig.Config)
		if options.Repository != "" {
			options.PrebuildRepositories = append(options.PrebuildRepositories, options.Repository)
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
				imageDetails, err := r.inspectImage(ctx, prebuildImage)
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

	dockerDriver, ok := r.Driver.(driver.DockerDriver)
	if !ok {
		return nil, fmt.Errorf("building is not supported with this driver")
	}

	return dockerDriver.BuildDevContainer(ctx, prebuildHash, parsedConfig, extendedBuildInfo, dockerfilePath, dockerfileContent, r.LocalWorkspaceFolder, options)
}
