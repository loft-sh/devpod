package docker

import (
	"bytes"
	"context"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider/types"
	"github.com/pkg/errors"
)

func (d *dockerProvider) WorkspaceDestroy(ctx context.Context, workspace *config.Workspace, options types.WorkspaceDestroyOptions) error {
	runner := d.newRunner(workspace)
	containerDetails, err := runner.FindDevContainer()
	if err != nil {
		return err
	} else if containerDetails == nil {
		return nil
	}

	status, err := d.WorkspaceStatus(ctx, workspace, types.WorkspaceStatusOptions{})
	if err != nil {
		return err
	} else if status == types.StatusNotFound {
		return nil
	}

	// stop before removing
	if status == types.StatusRunning {
		buf := &bytes.Buffer{}
		err = d.docker.Run([]string{"stop", "-t", "5", containerDetails.Id}, nil, buf, buf)
		if err != nil {
			return errors.Wrapf(err, "stop container %s", buf.String())
		}
	}

	// remove container if stopped
	buf := &bytes.Buffer{}
	err = d.docker.Run([]string{"rm", containerDetails.Id}, nil, buf, buf)
	if err != nil {
		return errors.Wrapf(err, "remove container %s", buf.String())
	}

	return nil
}
