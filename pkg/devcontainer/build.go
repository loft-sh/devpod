package devcontainer

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devpod/pkg/compose"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/feature"
	"github.com/loft-sh/devpod/pkg/devcontainer/metadata"
	"github.com/loft-sh/devpod/pkg/dockerfile"
	"github.com/loft-sh/devpod/pkg/driver"
	"github.com/loft-sh/devpod/pkg/driver/docker"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
)

func (r *runner) build(
	ctx context.Context,
	parsedConfig *config.SubstitutedConfig,
	substitutionContext *config.SubstitutionContext,
	options provider.BuildOptions,
) (*config.BuildInfo, error) {
	if isDockerFileConfig(parsedConfig.Config) {
		return r.buildAndExtendImage(ctx, parsedConfig, substitutionContext, options)
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

		overrideBuildImageName, _, imageMetadata, _, err := r.buildAndExtendDockerCompose(ctx, parsedConfig, substitutionContext, project, composeHelper, &composeService, composeGlobalArgs)
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

		// have a fallback value for PrebuildHash
		// we don't calculate prebuild hash on docker compose builds
		// let's use Images :tag then
		imageTag, err := r.getImageTag(ctx, imageDetails.ID)
		if err != nil {
			return nil, errors.Wrap(err, "inspect image")
		}

		return &config.BuildInfo{
			ImageDetails:  imageDetails,
			ImageMetadata: imageMetadata,
			ImageName:     overrideBuildImageName,
			PrebuildHash:  imageTag,
			RegistryCache: options.RegistryCache,
		}, nil
	}

	return r.extendImage(ctx, parsedConfig, substitutionContext, options)
}

func (r *runner) extendImage(
	ctx context.Context,
	parsedConfig *config.SubstitutedConfig,
	substitutionContext *config.SubstitutionContext,
	options provider.BuildOptions,
) (*config.BuildInfo, error) {
	imageBase := parsedConfig.Config.Image
	imageBuildInfo, err := r.getImageBuildInfoFromImage(ctx, substitutionContext, imageBase)
	if err != nil {
		return nil, errors.Wrap(err, "get image build info")
	}

	// get extend image build info
	extendedBuildInfo, err := feature.GetExtendedBuildInfo(substitutionContext, imageBuildInfo.Metadata, imageBuildInfo.User, imageBase, parsedConfig, r.Log, options.ForceBuild)
	if err != nil {
		return nil, errors.Wrap(err, "get extended build info")
	}

	// no need to build here
	if extendedBuildInfo == nil || extendedBuildInfo.FeaturesBuildInfo == nil {
		return &config.BuildInfo{
			ImageDetails:  imageBuildInfo.ImageDetails,
			ImageMetadata: extendedBuildInfo.MetadataConfig,
			ImageName:     imageBase,
			RegistryCache: options.RegistryCache,
		}, nil
	}

	// build the image
	return r.buildImage(ctx, parsedConfig, substitutionContext, imageBuildInfo, extendedBuildInfo, "", "", options)
}

func (r *runner) buildAndExtendImage(
	ctx context.Context,
	parsedConfig *config.SubstitutedConfig,
	substitutionContext *config.SubstitutionContext,
	options provider.BuildOptions,
) (*config.BuildInfo, error) {
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
	if parsedConfig.Config.GetTarget() != "" {
		imageBase = parsedConfig.Config.GetTarget()
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
	imageBuildInfo, err := r.getImageBuildInfoFromDockerfile(substitutionContext, string(dockerFileContent), parsedConfig.Config.GetArgs(), parsedConfig.Config.GetTarget())
	if err != nil {
		return nil, errors.Wrap(err, "get image build info")
	}

	// get extend image build info
	extendedBuildInfo, err := feature.GetExtendedBuildInfo(substitutionContext, imageBuildInfo.Metadata, imageBuildInfo.User, imageBase, parsedConfig, r.Log, options.ForceBuild)
	if err != nil {
		return nil, errors.Wrap(err, "get extended build info")
	}

	// build the image
	return r.buildImage(ctx, parsedConfig, substitutionContext, imageBuildInfo, extendedBuildInfo, dockerFilePath, string(dockerFileContent), options)
}

func (r *runner) getDockerfilePath(parsedConfig *config.DevContainerConfig) (string, error) {
	if parsedConfig.Origin == "" {
		return "", fmt.Errorf("couldn't find path where config was loaded from")
	}

	configFileDir := filepath.Dir(parsedConfig.Origin)

	dockerfilePath := filepath.Join(configFileDir, parsedConfig.GetDockerfile())
	_, err := os.Stat(dockerfilePath)
	if err != nil {
		return "", fmt.Errorf("couldn't find Dockerfile at %s", dockerfilePath)
	}

	return dockerfilePath, nil
}

func (r *runner) getImageBuildInfoFromImage(ctx context.Context, substitutionContext *config.SubstitutionContext, imageName string) (*config.ImageBuildInfo, error) {
	imageDetails, err := r.inspectImage(ctx, imageName)
	if err != nil {
		return nil, err
	}

	user := "root"
	if imageDetails.Config.User != "" {
		user = imageDetails.Config.User
	}

	imageMetadata, err := metadata.GetImageMetadata(imageDetails, substitutionContext, r.Log)
	if err != nil {
		return nil, errors.Wrap(err, "get image metadata")
	}

	return &config.ImageBuildInfo{
		ImageDetails: imageDetails,
		User:         user,
		Metadata:     imageMetadata,
	}, nil
}

func (r *runner) getImageBuildInfoFromDockerfile(substitutionContext *config.SubstitutionContext, dockerFileContent string, buildArgs map[string]string, target string) (*config.ImageBuildInfo, error) {
	parsedDockerfile, err := dockerfile.Parse(dockerFileContent)
	if err != nil {
		return nil, errors.Wrap(err, "parse dockerfile")
	}

	// Check that the build target specified in the devcontainer.json exists in the Dockerfile
	if target != "" && parsedDockerfile.StagesByTarget != nil {
		_, ok := parsedDockerfile.StagesByTarget[target]
		if !ok {
			return nil, fmt.Errorf("build target does not exist")
		}
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
	imageMetadataConfig, err := metadata.GetImageMetadata(imageDetails, substitutionContext, r.Log)
	if err != nil {
		return nil, errors.Wrap(err, "get image metadata")
	}

	return &config.ImageBuildInfo{
		Dockerfile: parsedDockerfile,
		User:       user,
		Metadata:   imageMetadataConfig,
	}, nil
}

func (r *runner) buildImage(
	ctx context.Context,
	parsedConfig *config.SubstitutedConfig,
	substitutionContext *config.SubstitutionContext,
	buildInfo *config.ImageBuildInfo,
	extendedBuildInfo *feature.ExtendedBuildInfo,
	dockerfilePath,
	dockerfileContent string,
	options provider.BuildOptions,
) (*config.BuildInfo, error) {
	targetArch, err := r.Driver.TargetArchitecture(ctx, r.ID)
	if err != nil {
		return nil, err
	}

	prebuildHash, err := config.CalculatePrebuildHash(parsedConfig.Config, options.Platform, targetArch, config.GetContextPath(parsedConfig.Config), dockerfilePath, dockerfileContent, buildInfo, r.Log)
	if err != nil {
		return nil, err
	}

	// check if there is a prebuild image
	if !options.ForceDockerless && !options.ForceBuild {
		devPodCustomizations := config.GetDevPodCustomizations(parsedConfig.Config)
		if options.Repository != "" {
			options.PrebuildRepositories = append(options.PrebuildRepositories, options.Repository)
		}
		options.PrebuildRepositories = append(options.PrebuildRepositories, devPodCustomizations.PrebuildRepository...)

		r.Log.Debugf("Try to find prebuild image %s in repositories %s", prebuildHash, strings.Join(options.PrebuildRepositories, ","))
		for _, prebuildRepo := range options.PrebuildRepositories {
			prebuildImage := prebuildRepo + ":" + prebuildHash
			img, err := image.GetImage(ctx, prebuildImage)
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
					RegistryCache: options.RegistryCache,
				}, nil
			} else if err != nil {
				r.Log.Debugf("Error trying to find prebuild image %s: %v", prebuildImage, err)
			}
		}
	}

	// check if we should fallback to dockerless
	dockerDriver, ok := r.Driver.(driver.DockerDriver)
	if options.ForceDockerless || !ok {
		if r.WorkspaceConfig.Agent.Dockerless.Disabled == "true" {
			return nil, fmt.Errorf("cannot build devcontainer because driver is non-docker and dockerless fallback is disabled")
		}

		return dockerlessFallback(r.LocalWorkspaceFolder, substitutionContext.ContainerWorkspaceFolder, parsedConfig, buildInfo, extendedBuildInfo, dockerfileContent, options)
	}

	return dockerDriver.BuildDevContainer(ctx, prebuildHash, parsedConfig, extendedBuildInfo, dockerfilePath, dockerfileContent, r.LocalWorkspaceFolder, options)
}

func dockerlessFallback(
	localWorkspaceFolder,
	containerWorkspaceFolder string,
	parsedConfig *config.SubstitutedConfig,
	buildInfo *config.ImageBuildInfo,
	extendedBuildInfo *feature.ExtendedBuildInfo,
	dockerfileContent string,
	options provider.BuildOptions,
) (*config.BuildInfo, error) {
	contextPath := config.GetContextPath(parsedConfig.Config)
	devPodInternalFolder := filepath.Join(contextPath, config.DevPodContextFeatureFolder)
	err := os.MkdirAll(devPodInternalFolder, 0755)
	if err != nil {
		return nil, fmt.Errorf("create devpod folder: %w", err)
	}

	// build dockerfile
	devPodDockerfile, err := docker.RewriteDockerfile(dockerfileContent, extendedBuildInfo)
	if err != nil {
		return nil, fmt.Errorf("rewrite dockerfile: %w", err)
	} else if devPodDockerfile == "" {
		devPodDockerfile = filepath.Join(devPodInternalFolder, "Dockerfile-without-features")
		err = os.WriteFile(devPodDockerfile, []byte(dockerfileContent), 0600)
		if err != nil {
			return nil, fmt.Errorf("write devpod dockerfile: %w", err)
		}
	}

	// get build args and target
	containerContext, containerDockerfile := getContainerContextAndDockerfile(localWorkspaceFolder, containerWorkspaceFolder, contextPath, devPodDockerfile)
	buildArgs, target := docker.GetBuildArgsAndTarget(parsedConfig, extendedBuildInfo)
	return &config.BuildInfo{
		ImageMetadata: extendedBuildInfo.MetadataConfig,
		Dockerless: &config.BuildInfoDockerless{
			Context:    containerContext,
			Dockerfile: containerDockerfile,

			BuildArgs: buildArgs,
			Target:    target,

			User: buildInfo.User,
		},
		RegistryCache: options.RegistryCache,
	}, nil
}

func getContainerContextAndDockerfile(localWorkspaceFolder, containerWorkspaceFolder, contextPath, devPodDockerfile string) (string, string) {
	prefixPath := path.Clean(filepath.ToSlash(localWorkspaceFolder))
	containerContext := path.Join(containerWorkspaceFolder, strings.TrimPrefix(path.Clean(filepath.ToSlash(contextPath)), prefixPath))
	containerDockerfile := path.Join(containerWorkspaceFolder, strings.TrimPrefix(path.Clean(filepath.ToSlash(devPodDockerfile)), prefixPath))
	return containerContext, containerDockerfile
}
