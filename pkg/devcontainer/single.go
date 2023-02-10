package devcontainer

import (
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/metadata"
	"github.com/pkg/errors"
)

func (r *Runner) runSingleContainer(parsedConfig *config.SubstitutedConfig, workspaceMount string) error {
	labels := r.getLabels()
	containerDetails, err := r.Docker.FindDevContainer(labels)
	if err != nil {
		return errors.Wrap(err, "find dev container")
	}

	// does the container already exist?
	var mergedConfig *config.MergedDevContainerConfig
	if containerDetails != nil {
		// start container if not running
		if containerDetails.State.Status != "running" {
			err = r.Docker.StartContainer(containerDetails.Id, labels)
			if err != nil {
				return err
			}
		}

		imageMetadataConfig, err := metadata.GetImageMetadataFromContainer(containerDetails, r.SubstitutionContext, r.Log)
		if err != nil {
			return err
		}

		mergedConfig, err = config.MergeConfiguration(parsedConfig.Config, imageMetadataConfig.Config)
		if err != nil {
			return errors.Wrap(err, "merge config")
		}
	} else {
		// we need to build container
		buildInfo, err := r.build(parsedConfig)
		if err != nil {
			return errors.Wrap(err, "build image")
		}

		mergedConfig, err = config.MergeConfiguration(parsedConfig.Config, buildInfo.ImageMetadata.Config)
		if err != nil {
			return errors.Wrap(err, "merge config")
		}

		// TODO: for non build images, add metadata label to image here during start

		err = r.startDevContainer(parsedConfig.Config, mergedConfig, buildInfo.ImageName, workspaceMount, labels, buildInfo.ImageDetails)
		if err != nil {
			return errors.Wrap(err, "start dev container")
		}

		// TODO: wait here a bit for correct startup?

		// get container details
		containerDetails, err = r.Docker.FindDevContainer(labels)
		if err != nil {
			return err
		}
	}

	// setup container
	//return r.setupContainer(containerDetails, mergedConfig)

	return nil
}
