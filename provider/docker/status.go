package docker

import (
	"context"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider/types"
	"strings"
)

func (d *dockerProvider) WorkspaceStatus(ctx context.Context, workspace *config.Workspace, options types.WorkspaceStatusOptions) (types.Status, error) {
	runner := d.newRunner(workspace)
	containerDetails, err := runner.FindDevContainer()
	if err != nil {
		return types.StatusNotFound, err
	} else if containerDetails == nil {
		return types.StatusNotFound, nil
	}

	status := strings.ToLower(containerDetails.State.Status)
	if status == "running" {
		return types.StatusRunning, nil
	} else if status == "paused" {
		return types.StatusBusy, nil
	} else if status == "exited" {
		return types.StatusStopped, nil
	}

	return types.StatusBusy, nil
}
