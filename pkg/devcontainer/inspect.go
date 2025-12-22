package devcontainer

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver"
	"github.com/loft-sh/devpod/pkg/image"
)

func (r *runner) inspectImage(ctx context.Context, imageName string) (*config.ImageDetails, error) {
	dockerDriver, ok := r.Driver.(driver.DockerDriver)
	if ok {
		return dockerDriver.InspectImage(ctx, imageName)
	}

	// Get target architecture from the driver
	targetArch, err := r.Driver.TargetArchitecture(ctx, r.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target architecture: %w", err)
	}

	// Use architecture-specific image config retrieval
	imageConfig, _, err := image.GetImageConfigForArch(ctx, imageName, targetArch, r.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to get image config for architecture %s: %w", targetArch, err)
	}

	return &config.ImageDetails{
		ID: imageName,
		Config: config.ImageDetailsConfig{
			User:       imageConfig.Config.User,
			Env:        imageConfig.Config.Env,
			Labels:     imageConfig.Config.Labels,
			Entrypoint: imageConfig.Config.Entrypoint,
			Cmd:        imageConfig.Config.Cmd,
		},
	}, nil
}

func (r *runner) getImageTag(ctx context.Context, imageID string) (string, error) {
	dockerDriver, ok := r.Driver.(driver.DockerDriver)
	if ok {
		return dockerDriver.GetImageTag(ctx, imageID)
	}

	return "", nil
}
