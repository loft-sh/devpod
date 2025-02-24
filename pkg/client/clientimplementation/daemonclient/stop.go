package daemonclient

import (
	"context"
	"fmt"
	"os"

	clientpkg "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/remotecommand"
)

func (c *client) Stop(ctx context.Context, opt clientpkg.StopOptions) error {
	c.m.Lock()
	defer c.m.Unlock()

	baseClient, err := c.initPlatformClient(ctx)
	if err != nil {
		return err
	}
	workspace, err := platform.FindInstance(ctx, baseClient, c.workspace.UID)
	if err != nil {
		return err
	} else if workspace == nil {
		return fmt.Errorf("couldn't find workspace")
	}

	conn, err := platform.DialInstance(baseClient, workspace, "stop", platform.URLOptions(opt), c.log)
	if err != nil {
		return err
	}

	_, err = remotecommand.ExecuteConn(ctx, conn, os.Stdin, os.Stdout, os.Stderr, c.log)
	if err != nil {
		return fmt.Errorf("error executing: %w", err)
	}

	return nil
}
