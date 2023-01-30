package docker

import (
	"bytes"
	"context"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider/types"
	"github.com/pkg/errors"
)

func (d *dockerProvider) WorkspaceStart(ctx context.Context, workspace *config.Workspace, options types.WorkspaceStartOptions) error {
	runner := d.newRunner(workspace)
	containerDetails, err := runner.FindDevContainer()
	if err != nil {
		return err
	} else if containerDetails == nil {
		return nil
	}

	buf := &bytes.Buffer{}
	err = d.docker.Run([]string{"start", containerDetails.Id}, nil, buf, buf)
	if err != nil {
		return errors.Wrapf(err, "start container %s", buf.String())
	}

	return nil
}
