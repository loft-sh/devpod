package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/feature"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/devpod/pkg/random"
	"github.com/pkg/errors"
	"strings"
)

func (k *kubernetesDriver) PushDevContainer(ctx context.Context, image string) error {
	return fmt.Errorf("not supported")
}

func (k *kubernetesDriver) BuildDevContainer(
	ctx context.Context,
	parsedConfig *config.SubstitutedConfig,
	extendedBuildInfo *feature.ExtendedBuildInfo,
	dockerfilePath,
	dockerfileContent string,
	localWorkspaceFolder string,
	options config.BuildOptions,
) (*config.BuildInfo, error) {
	// get cluster architecture
	arch, err := k.getClusterArchitecture()
	if err != nil {
		return nil, err
	}

	prebuildHash, err := config.CalculatePrebuildHash(parsedConfig.Config, arch, dockerfileContent, k.Log)
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
		k.Log.Debugf("Try to find prebuild image %s in repositories %s", prebuildHash, strings.Join(options.PrebuildRepositories, ","))
		for _, prebuildRepo := range options.PrebuildRepositories {
			prebuildImage := prebuildRepo + ":" + prebuildHash
			img, err := image.GetImage(prebuildImage)
			if err == nil && img != nil {
				// prebuild image found
				k.Log.Infof("Found existing prebuilt image %s", prebuildImage)

				// inspect image
				imageDetails, err := k.InspectImage(ctx, prebuildImage)
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
				k.Log.Debugf("Error trying to find prebuild image %s: %v", prebuildImage, err)
			}
		}
	}

	// check if we shouldn't build
	if options.NoBuild {
		return nil, fmt.Errorf("you cannot build in this mode. Please run 'devpod up' to rebuild the container")
	}

	return nil, fmt.Errorf("you cannot build with this driver. Please use another driver to prebuild this image")
}

func (k *kubernetesDriver) getClusterArchitecture() (string, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := k.runCommand(context.TODO(), []string{"run", "-i", "devpod-" + random.String(6), "-q", "--rm", "--restart=Never", "--image", k.helperImage(), "--", "sh"}, strings.NewReader("uname -a; exit 0"), stdout, stderr)
	if err != nil {
		return "", fmt.Errorf("find out cluster architecture: %s %s %v", stdout.String(), stderr.String(), err)
	}

	unameOutput := stdout.String()
	if strings.Contains(unameOutput, "arm") || strings.Contains(unameOutput, "aarch") {
		return "arm64", nil
	}

	return "amd64", nil
}

func (k *kubernetesDriver) helperImage() string {
	if k.config.HelperImage != "" {
		return k.config.HelperImage
	}

	return "busybox"
}
