package devcontainer

import (
	"context"
	"strings"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/pkg/errors"
)

func (r *Runner) Delete(labels []string, deleteVolumes bool) error {
	if len(labels) == 0 {
		labels = r.getLabels()
	}
	containerDetails, err := r.Driver.FindDevContainer(context.TODO(), labels)
	if err != nil {
		return errors.Wrap(err, "find dev container")
	} else if containerDetails == nil {
		return nil
	}

	r.Log.Infof("Deleting devcontainer...")
	if isDockerCompose, projectName := getDockerComposeProject(containerDetails); isDockerCompose {
		err = r.deleteDockerCompose(projectName)
		if err != nil {
			return err
		}
	} else {
		if strings.ToLower(containerDetails.State.Status) == "running" {
			err = r.Driver.StopDevContainer(context.TODO(), containerDetails.ID)
			if err != nil {
				return err
			}
		}

		err = r.Driver.DeleteDevContainer(context.TODO(), containerDetails.ID, deleteVolumes)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) Stop() error {
	labels := r.getLabels()
	containerDetails, err := r.Driver.FindDevContainer(context.TODO(), labels)
	if err != nil {
		return errors.Wrap(err, "find dev container")
	} else if containerDetails == nil {
		return nil
	}

	if strings.ToLower(containerDetails.State.Status) == "running" {
		if isDockerCompose, projectName := getDockerComposeProject(containerDetails); isDockerCompose {
			err = r.stopDockerCompose(projectName)
			if err != nil {
				return err
			}
		} else {
			err = r.Driver.StopDevContainer(context.TODO(), containerDetails.ID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getDockerComposeProject(containerDetails *config.ContainerDetails) (bool, string) {
	if projectName, ok := containerDetails.Config.Labels["com.docker.compose.project"]; ok {
		return true, projectName
	}

	return false, ""
}
