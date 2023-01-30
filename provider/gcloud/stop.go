package gcloud

import (
	"context"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider/types"
	"github.com/pkg/errors"
)

func (g *gcloudProvider) Stop(ctx context.Context, workspace *config.Workspace, options types.StopOptions) error {
	name := getName(workspace)
	args := []string{
		"compute",
		"instances",
		"stop",
		name,
		"--project=" + g.Config.Project,
		"--zone=" + g.Config.Zone,
		"--async",
	}

	g.Log.Infof("Stopping VM Instance %s...", name)
	_, err := g.output(ctx, args...)
	if err != nil {
		return errors.Wrapf(err, "stop vm")
	}

	g.Log.Infof("Successfully stopped VM instance %s", name)
	return nil
}
