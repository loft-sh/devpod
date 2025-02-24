package daemonclient

import (
	"context"
	"fmt"
	"time"

	clientpkg "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/platform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *client) Delete(ctx context.Context, opt clientpkg.DeleteOptions) error {
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

	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}
	var gracePeriod *time.Duration
	if opt.GracePeriod != "" {
		duration, err := time.ParseDuration(opt.GracePeriod)
		if err == nil {
			gracePeriod = &duration
		}
	}

	// kill the command after the grace period
	if gracePeriod != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *gracePeriod)
		defer cancel()
	}

	err = managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(workspace.Namespace).Delete(ctx, workspace.Name, metav1.DeleteOptions{})
	if err != nil {
		if !opt.Force {
			return fmt.Errorf("delete workspace: %w", err)
		}

		c.log.Errorf("Error deleting workspace: %v", err)
	}

	return clientimplementation.DeleteWorkspaceFolder(c.workspace.Context, c.workspace.ID, c.workspace.SSHConfigPath, c.log)
}
