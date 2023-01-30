package gcloud

import (
	"context"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider/types"
	"github.com/pkg/errors"
)

func (g *gcloudProvider) Destroy(ctx context.Context, workspace *config.Workspace, options types.DestroyOptions) error {
	name := getName(workspace)
	args := []string{
		"compute",
		"instances",
		"delete",
		name,
		"--project=" + g.Config.Project,
		"--zone=" + g.Config.Zone,
	}

	g.Log.Infof("Deleting VM Instance %s...", name)
	_, err := g.output(ctx, args...)
	if err != nil {
		return errors.Wrapf(err, "destroy vm")
	}

	g.Log.Infof("Successfully deleted VM instance %s", name)
	return nil
}
