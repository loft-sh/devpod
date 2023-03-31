package devcontainer

import (
	"encoding/json"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/metadata"
	"github.com/pkg/errors"
)

func (r *Runner) runSingleContainer(parsedConfig *config.SubstitutedConfig, workspaceMount string, options UpOptions) (*config.Result, error) {
	labels := r.getLabels()
	if options.Recreate {
		err := r.Delete(labels)
		if err != nil {
			return nil, errors.Wrap(err, "delete devcontainer")
		}
	}

	containerDetails, err := r.Driver.FindDevContainer(labels)
	if err != nil {
		return nil, errors.Wrap(err, "find dev container")
	}

	// does the container already exist?
	var mergedConfig *config.MergedDevContainerConfig
	if containerDetails != nil {
		// start container if not running
		if containerDetails.State.Status != "running" {
			err = r.Driver.StartDevContainer(containerDetails.Id, labels)
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
		buildInfo, err := r.build(parsedConfig, config.BuildOptions{
			PrebuildRepositories: options.PrebuildRepositories,
			NoBuild:              options.NoBuild,
			ForceRebuild:         options.ForceBuild,
		})
		if err != nil {
			return nil, errors.Wrap(err, "build image")
		}

		mergedConfig, err = config.MergeConfiguration(parsedConfig.Config, buildInfo.ImageMetadata.Config)
		if err != nil {
			return nil, errors.Wrap(err, "merge config")
		}

		// have we built the image?
		if buildInfo.ImageName == parsedConfig.Config.Image {
			// add metadata as label here
			marshalled, err := json.Marshal(buildInfo.ImageMetadata.Raw)
			if err != nil {
				return nil, errors.Wrap(err, "marshal config")
			}

			labels = append(labels, metadata.ImageMetadataLabel+"="+string(marshalled))
		}

		err = r.Driver.RunDevContainer(parsedConfig.Config, mergedConfig, buildInfo.ImageName, workspaceMount, labels, r.WorkspaceConfig.Workspace.IDE.Name, r.WorkspaceConfig.Workspace.IDE.Options, buildInfo.ImageDetails)
		if err != nil {
			return nil, errors.Wrap(err, "start dev container")
		}

		//TODO: wait here a bit for correct startup?

		// get container details
		containerDetails, err = r.Driver.FindDevContainer(labels)
		if err != nil {
			return nil, err
		}
	}

	// substitute config with container env
	newMergedConfig := &config.MergedDevContainerConfig{}
	err = config.SubstituteContainerEnv(config.ListToObject(containerDetails.Config.Env), mergedConfig, newMergedConfig)
	if err != nil {
		return nil, errors.Wrap(err, "substitute container env")
	}

	// setup container
	err = r.setupContainer(containerDetails, mergedConfig)
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
