package daemonclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	clientpkg "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/remotecommand"
	"github.com/loft-sh/log"
	corev1 "k8s.io/api/core/v1"
)

func (c *client) Up(ctx context.Context, opt clientpkg.UpOptions) error {
	baseClient, err := c.initPlatformClient(ctx)
	if err != nil {
		return err
	}

	instance, err := platform.FindInstance(ctx, baseClient, c.workspace.UID)
	if err != nil {
		return err
	} else if instance == nil {
		return fmt.Errorf("workspace %s not found. Looks like it does not exist anymore and you can delete it", c.workspace.ID)
	}

	// Log current workspace information. This is both useful to the user to understand the workspace configuration
	// and to us when we receive troubleshooting logs
	printInstanceInfo(instance, c.log)

	if instance.Spec.TemplateRef != nil && templateUpdateRequired(instance) {
		c.log.Info("Template update required")
		oldInstance := instance.DeepCopy()
		instance.Spec.TemplateRef.SyncOnce = true

		instance, err = platform.UpdateInstance(ctx, baseClient, oldInstance, instance, c.log)
		if err != nil {
			return fmt.Errorf("update instance: %w", err)
		}
		c.log.Info("Successfully updated template")
	}

	conn, err := platform.DialInstance(baseClient, instance, "up", platform.URLOptions(opt), c.log)
	if err != nil {
		return err
	}

	_, err = remotecommand.ExecuteConn(ctx, conn, opt.Stdin, opt.Stdout, os.Stderr, c.log)
	if err != nil {
		return fmt.Errorf("error executing: %w", err)
	}
	return nil
}

func templateUpdateRequired(instance *managementv1.DevPodWorkspaceInstance) bool {
	var templateResolved, templateChangesAvailable bool
	for _, condition := range instance.Status.Conditions {
		if condition.Type == storagev1.InstanceTemplateResolved {
			templateResolved = condition.Status == corev1.ConditionTrue
			continue
		}

		if condition.Type == storagev1.InstanceTemplateSynced {
			templateChangesAvailable = condition.Status == corev1.ConditionFalse &&
				condition.Reason == "TemplateChangesAvailable"
			continue
		}
	}

	return !templateResolved || templateChangesAvailable
}

func printInstanceInfo(instance *managementv1.DevPodWorkspaceInstance, log log.Logger) {
	workspaceConfig, _ := json.Marshal(struct {
		Runner     storagev1.RunnerRef
		Template   *storagev1.TemplateRef
		Parameters string
	}{
		Runner:     instance.Spec.RunnerRef,
		Template:   instance.Spec.TemplateRef,
		Parameters: instance.Spec.Parameters,
	})
	log.Info("Starting pro workspace with configuration", string(workspaceConfig))
}
