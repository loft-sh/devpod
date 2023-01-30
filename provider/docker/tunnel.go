package docker

import (
	"context"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider/types"
	"github.com/loft-sh/devpod/pkg/token"
	"github.com/pkg/errors"
)

func (d *dockerProvider) WorkspaceTunnel(ctx context.Context, workspace *config.Workspace, options types.WorkspaceTunnelOptions) error {
	runner := d.newRunner(workspace)
	containerDetails, err := runner.FindDevContainer()
	if err != nil {
		return err
	} else if containerDetails == nil {
		return nil
	}

	tok, err := token.GenerateWorkspaceToken(workspace.ID)
	if err != nil {
		return errors.Wrap(err, "generate token")
	}

	err = runner.Docker.Tunnel(runner.AgentDownloadURL, containerDetails.Id, tok, options.Stdin, options.Stdout, options.Stderr)
	if err != nil {
		return err
	}

	return nil
}
