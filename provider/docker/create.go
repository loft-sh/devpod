package docker

import (
	"context"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider/types"
)

func (d *dockerProvider) WorkspaceCreate(ctx context.Context, workspace *config.Workspace, options types.WorkspaceCreateOptions) error {
	err := d.newRunner(workspace).Up()
	if err != nil {
		return err
	}

	return nil
}
