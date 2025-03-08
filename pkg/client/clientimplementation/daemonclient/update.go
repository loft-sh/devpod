package daemonclient

import (
	"context"
	"fmt"

	"github.com/loft-sh/log/terminal"
)

func (c *client) updateInstance(ctx context.Context) error {
	baseClient, err := c.initPlatformClient(ctx)
	if err != nil {
		return err
	}
	if !terminal.IsTerminalIn {
		return fmt.Errorf("unable to update instance through CLI if stdin is not a terminal")
	}

	oldInstance, err := c.localClient.GetWorkspace(ctx, c.workspace.UID)
	if err != nil {
		return err
	}
	if oldInstance == nil {
		return fmt.Errorf("unable to find old workspace instance")
	}
	newInstance, err := updateInstanceInteractive(ctx, baseClient, oldInstance, c.log)
	if err != nil {
		return err
	}

	_, err = c.localClient.UpdateWorkspace(ctx, newInstance)
	return err
}
