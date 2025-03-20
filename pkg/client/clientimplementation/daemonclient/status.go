package daemonclient

import (
	"context"
	"fmt"

	clientpkg "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/platform"
)

func (c *client) Status(ctx context.Context, opt clientpkg.StatusOptions) (clientpkg.Status, error) {
	c.m.Lock()
	defer c.m.Unlock()

	status := clientpkg.Status(clientpkg.StatusNotFound)
	baseClient, err := c.initPlatformClient(ctx)
	if err != nil {
		return status, err
	}

	instance, err := platform.FindInstance(ctx, baseClient, c.workspace.UID)
	if err != nil {
		return status, err
	} else if instance == nil {
		return status, fmt.Errorf("couldn't find workspace")
	}

	return clientpkg.Status(instance.Status.LastWorkspaceStatus), nil
}
