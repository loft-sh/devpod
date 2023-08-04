package devcontainer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/metadata"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
)

func (r *Runner) runSingleContainer(ctx context.Context, parsedConfig *config.SubstitutedConfig, workspaceMount string, options UpOptions) (*config.Result, error) {
	labels := r.getLabels()
	containerDetails, err := r.Driver.FindDevContainer(ctx, labels)
	if err != nil {
		return nil, fmt.Errorf("find dev container: %w", err)
	}

	// does the container already exist?
	var mergedConfig *config.MergedDevContainerConfig
	if !options.Recreate && containerDetails != nil {
		// start container if not running
		if strings.ToLower(containerDetails.State.Status) != "running" {
			err = r.Driver.StartDevContainer(ctx, containerDetails.ID, labels)
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
		// we need to build container
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
			err := r.Delete(ctx, labels, false)
			if err != nil {
				return nil, errors.Wrap(err, "delete devcontainer")
			}
		}

		// merge configuration
		mergedConfig, err = config.MergeConfiguration(parsedConfig.Config, buildInfo.ImageMetadata.Config)
		if err != nil {
			return nil, errors.Wrap(err, "merge config")
		}

		// add metadata as label here
		marshalled, err := json.Marshal(buildInfo.ImageMetadata.Raw)
		if err != nil {
			return nil, errors.Wrap(err, "marshal config")
		}
		labels = append(labels, metadata.ImageMetadataLabel+"="+string(marshalled))

		// run dev container
		err = r.Driver.RunDevContainer(ctx, parsedConfig.Config, mergedConfig, buildInfo.ImageName, workspaceMount, labels, r.WorkspaceConfig.Workspace.IDE.Name, r.WorkspaceConfig.Workspace.IDE.Options, buildInfo.ImageDetails)
		if err != nil {
			return nil, errors.Wrap(err, "start dev container")
		}

		// TODO: wait here a bit for correct startup?

		// get container details
		containerDetails, err = r.Driver.FindDevContainer(ctx, labels)
		if err != nil {
			return nil, err
		}
	}

	// set remoteenv
	if mergedConfig.RemoteEnv == nil {
		mergedConfig.RemoteEnv = make(map[string]string)
	}
	if _, ok := mergedConfig.RemoteEnv["PATH"]; !ok {
		mergedConfig.RemoteEnv["PATH"] = "${containerEnv:PATH}"
	}

	// substitute config with container env
	newMergedConfig := &config.MergedDevContainerConfig{}
	err = config.SubstituteContainerEnv(config.ListToObject(containerDetails.Config.Env), mergedConfig, newMergedConfig)
	if err != nil {
		return nil, errors.Wrap(err, "substitute container env")
	}

	// setup container
	err = r.setupContainer(containerDetails, newMergedConfig, options)
	if err != nil {
		return nil, errors.Wrap(err, "setup container")
	}

	// return result
	return &config.Result{
		ContainerDetails:    containerDetails,
		MergedConfig:        newMergedConfig,
		SubstitutionContext: r.SubstitutionContext,
	}, nil
}
