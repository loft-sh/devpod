package daemonclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/apiserver/pkg/builders"
	clientpkg "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *client) Up(ctx context.Context, opt clientpkg.UpOptions) (*config.Result, error) {
	baseClient, err := c.initPlatformClient(ctx)
	if err != nil {
		return nil, err
	}

	instance, err := platform.FindInstance(ctx, baseClient, c.workspace.UID)
	if err != nil {
		return nil, err
	} else if instance == nil {
		return nil, fmt.Errorf("workspace %s not found. Looks like it does not exist anymore and you can delete it", c.workspace.ID)
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
			return nil, fmt.Errorf("update instance: %w", err)
		}
		c.log.Info("Successfully updated template")
	}

	// encode options
	rawOptions, _ := json.Marshal(opt)
	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, fmt.Errorf("error getting management client: %w", err)
	}

	// create up task
	task, err := managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(instance.Namespace).Up(ctx, instance.Name, &managementv1.DevPodWorkspaceInstanceUp{
		Spec: managementv1.DevPodWorkspaceInstanceUpSpec{
			Options: string(rawOptions),
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating up: %w", err)
	} else if task.Status.TaskID == "" {
		return nil, fmt.Errorf("no up task id returned from server")
	}

	// stream logs
	exitCode, err := printLogs(ctx, managementClient, instance, task.Status.TaskID, os.Stdout, os.Stderr, c.log)
	if err != nil {
		return nil, fmt.Errorf("error printing logs: %w", err)
	} else if exitCode != 0 {
		return nil, fmt.Errorf("up failed with exit code %d", exitCode)
	}

	// get result
	tasks := &managementv1.DevPodWorkspaceInstanceTasks{}
	err = managementClient.Loft().ManagementV1().RESTClient().Get().
		Namespace(instance.Namespace).
		Resource("devpodworkspaceinstances").
		Name(instance.Name).
		SubResource("tasks").
		VersionedParams(&managementv1.DevPodWorkspaceInstanceTasksOptions{
			TaskID: task.Status.TaskID,
		}, builders.ParameterCodec).
		Do(ctx).
		Into(tasks)
	if err != nil {
		return nil, fmt.Errorf("error getting up result: %w", err)
	} else if len(tasks.Tasks) == 0 || tasks.Tasks[0].Result == "" {
		return nil, fmt.Errorf("up result not found")
	} else if len(tasks.Tasks) > 1 {
		return nil, fmt.Errorf("multiple up results found")
	}

	// decompress result
	compressedResult, err := compress.Decompress(tasks.Tasks[0].Result)
	if err != nil {
		return nil, fmt.Errorf("error decompressing up result: %w", err)
	}

	// unmarshal result
	result := &config.Result{}
	err = json.Unmarshal([]byte(compressedResult), result)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling up result: %w", err)
	}

	// return result
	return result, nil
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
		Cluster    storagev1.ClusterRef
		Template   *storagev1.TemplateRef
		Parameters string
	}{
		Cluster:    instance.Spec.ClusterRef,
		Template:   instance.Spec.TemplateRef,
		Parameters: instance.Spec.Parameters,
	})
	log.Info("Starting pro workspace with configuration", string(workspaceConfig))
}
