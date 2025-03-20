package daemonclient

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	clientpkg "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/platform"
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

	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	rawOptions, _ := json.Marshal(opt)
	retStop, err := managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(workspace.Namespace).Stop(ctx, workspace.Name, &managementv1.DevPodWorkspaceInstanceStop{
		Spec: managementv1.DevPodWorkspaceInstanceStopSpec{
			Options: string(rawOptions),
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("error stopping workspace: %w", err)
	} else if retStop.Status.TaskID == "" {
		return fmt.Errorf("no stop task id returned from server")
	}

	_, err = observeTask(ctx, managementClient, workspace, retStop.Status.TaskID, c.log)
	if err != nil {
		return fmt.Errorf("stop: %w", err)
	}

	return nil
}
