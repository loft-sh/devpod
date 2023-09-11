package devcontainer

import (
	"context"
	"strings"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/pkg/errors"
)

func (r *runner) Delete(ctx context.Context) error {
	containerDetails, err := r.Driver.FindDevContainer(ctx, r.ID)
	if err != nil {
		return errors.Wrap(err, "find dev container")
	} else if containerDetails == nil {
		return nil
	}

	r.Log.Infof("Deleting devcontainer...")
	if isDockerCompose, projectName := getDockerComposeProject(containerDetails); isDockerCompose {
		err = r.deleteDockerCompose(ctx, projectName)
		if err != nil {
			return err
		}
	} else {
		if strings.ToLower(containerDetails.State.Status) == "running" {
			err = r.Driver.StopDevContainer(ctx, r.ID)
			if err != nil {
				return err
			}
		}

		err = r.Driver.DeleteDevContainer(ctx, r.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *runner) Stop(ctx context.Context) error {
	containerDetails, err := r.Driver.FindDevContainer(ctx, r.ID)
	if err != nil {
		return errors.Wrap(err, "find dev container")
	} else if containerDetails == nil {
		return nil
	}

	if strings.ToLower(containerDetails.State.Status) == "running" {
		if isDockerCompose, projectName := getDockerComposeProject(containerDetails); isDockerCompose {
			err = r.stopDockerCompose(ctx, projectName)
			if err != nil {
				return err
			}
		} else {
			err = r.Driver.StopDevContainer(ctx, r.ID)
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
