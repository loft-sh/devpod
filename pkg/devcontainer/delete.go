package devcontainer

import (
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/pkg/errors"
	"strings"
)

func (r *Runner) Delete(labels []string) error {
	if len(labels) == 0 {
		labels = r.getLabels()
	}
	containerDetails, err := r.Docker.FindDevContainer(labels)
	if err != nil {
		return errors.Wrap(err, "find dev container")
	} else if containerDetails == nil {
		return nil
	}

	r.Log.Infof("Deleting devcontainer...")
	if isDockerCompose, projectName := getDockerComposeProject(containerDetails); isDockerCompose {
		err = r.Compose.Remove(projectName)
		if err != nil {
			return err
		}
	} else {
		if strings.ToLower(containerDetails.State.Status) == "running" {
			err = r.Docker.Stop(containerDetails.Id)
			if err != nil {
				return err
			}
		}

		err = r.Docker.Remove(containerDetails.Id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) Stop() error {
	labels := r.getLabels()
	containerDetails, err := r.Docker.FindDevContainer(labels)
	if err != nil {
		return errors.Wrap(err, "find dev container")
	} else if containerDetails == nil {
		return nil
	}

	if strings.ToLower(containerDetails.State.Status) == "running" {
		if isDockerCompose, projectName := getDockerComposeProject(containerDetails); isDockerCompose {
			err = r.Compose.Stop(projectName)
			if err != nil {
				return err
			}
		} else {
			err = r.Docker.Stop(containerDetails.Id)
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
