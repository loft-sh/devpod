package gcloud

import (
	"context"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider/types"
	"github.com/pkg/errors"
)

func (g *gcloudProvider) Start(ctx context.Context, workspace *config.Workspace, options types.StartOptions) error {
	name := getName(workspace)
	args := []string{
		"compute",
		"instances",
		"start",
		name,
		"--project=" + g.Config.Project,
		"--zone=" + g.Config.Zone,
	}

	g.Log.Infof("Starting VM Instance %s...", name)
	_, err := g.output(ctx, args...)
	if err != nil {
		return errors.Wrapf(err, "start vm")
	}

	g.Log.Infof("Successfully started VM instance %s", name)
	return nil
}
