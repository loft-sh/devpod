package devcontainer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/metadata"
	"github.com/loft-sh/devpod/pkg/driver"
	"github.com/loft-sh/devpod/pkg/driver/docker"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
)

func (r *Runner) runSingleContainer(ctx context.Context, parsedConfig *config.SubstitutedConfig, options UpOptions) (*config.Result, error) {
	containerDetails, err := r.Driver.FindDevContainer(ctx, r.ID)
	if err != nil {
		return nil, fmt.Errorf("find dev container: %w", err)
	}

	// does the container already exist?
	var (
		mergedConfig *config.MergedDevContainerConfig
	)
	if !options.Recreate && containerDetails != nil {
		// start container if not running
		if strings.ToLower(containerDetails.State.Status) != "running" {
			err = r.Driver.StartDevContainer(ctx, r.ID)
			if err != nil {
				return nil, err
			}
		}

		imageMetadataConfig, err := metadata.GetImageMetadataFromContainer(containerDetails, r.SubstitutionContext, r.Log)
		if err != nil {
			return nil, err
		}

		mergedConfig, err = config.MergeConfiguration(parsedConfig.Config, imageMetadataConfig.Config)
		if err != nil {
			return nil, errors.Wrap(err, "merge config")
		}
	} else {
		// we need to build the container
		buildInfo, err := r.build(ctx, parsedConfig, config.BuildOptions{
			CLIOptions: provider2.CLIOptions{
				PrebuildRepositories: options.PrebuildRepositories,
			},
			NoBuild: options.NoBuild,
		})
		if err != nil {
			return nil, errors.Wrap(err, "build image")
		}

		// delete container on recreation
		if options.Recreate {
			err := r.Delete(ctx, false)
			if err != nil {
				return nil, errors.Wrap(err, "delete devcontainer")
			}
		}

		// merge configuration
		mergedConfig, err = config.MergeConfiguration(parsedConfig.Config, buildInfo.ImageMetadata.Config)
		if err != nil {
			return nil, errors.Wrap(err, "merge config")
		}

		// run dev container
		err = r.runContainer(ctx, parsedConfig, mergedConfig, buildInfo)
		if err != nil {
			return nil, errors.Wrap(err, "start dev container")
		}

		// TODO: wait here a bit for correct startup?

		// get from build info
		containerDetails, err = r.Driver.FindDevContainer(ctx, r.ID)
		if err != nil {
			return nil, fmt.Errorf("find dev container: %w", err)
		}
	}

	// setup container
	return r.setupContainer(ctx, containerDetails, mergedConfig)
}

func (r *Runner) runContainer(
	ctx context.Context,
	parsedConfig *config.SubstitutedConfig,
	mergedConfig *config.MergedDevContainerConfig,
	buildInfo *config.BuildInfo,
) error {
	// add metadata as label here
	marshalled, err := json.Marshal(buildInfo.ImageMetadata.Raw)
	if err != nil {
		return errors.Wrap(err, "marshal config")
	}

	// build labels
	labels := []string{metadata.ImageMetadataLabel + "=" + string(marshalled)}

	// check if docker
	dockerDriver, ok := r.Driver.(driver.DockerDriver)
	if ok {
		return dockerDriver.RunDockerDevContainer(ctx, r.ID, parsedConfig.Config, mergedConfig, buildInfo.ImageName, r.SubstitutionContext.WorkspaceMount, labels, r.WorkspaceConfig.Workspace.IDE.Name, r.WorkspaceConfig.Workspace.IDE.Options, buildInfo.ImageDetails)
	}

	// build run options for regular driver
	entrypoint, cmd := docker.GetContainerEntrypointAndArgs(mergedConfig, buildInfo.ImageDetails)
	workspaceMountParsed := config.ParseMount(r.SubstitutionContext.WorkspaceMount)
	return r.Driver.RunDevContainer(ctx, r.ID, &driver.RunOptions{
		Image:          buildInfo.ImageName,
		User:           buildInfo.ImageDetails.Config.User,
		Entrypoint:     entrypoint,
		Cmd:            cmd,
		Env:            mergedConfig.ContainerEnv,
		CapAdd:         mergedConfig.CapAdd,
		Labels:         labels,
		Privileged:     mergedConfig.Privileged,
		WorkspaceMount: &workspaceMountParsed,
		Mounts:         mergedConfig.Mounts,
	})
}
