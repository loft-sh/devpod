package daemonclient

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/apiserver/pkg/builders"
	clientpkg "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	devpodlog "github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/kube"
	"github.com/loft-sh/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	_ "github.com/loft-sh/api/v4/pkg/apis/management/install" // Install the management group to ensure the option types are registered
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

	// prompt user to attach to active task or start new one
	c.log.Debug("Check active up task")
	activeUpTask, err := findActiveUpTask(ctx, managementClient, instance)
	if err != nil {
		return nil, fmt.Errorf("find active up task: %w", err)
	}

	// if we have an active up task, cancel it before creating a new one
	if activeUpTask != nil {
		c.log.Warnf("Found active up task %s, attempting to cancel it", activeUpTask.ID)
		_, err = managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(instance.Namespace).Cancel(ctx, instance.Name, &managementv1.DevPodWorkspaceInstanceCancel{
			TaskID: activeUpTask.ID,
		}, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("cancel task: %w", err)
		}
	}

	// create up task
	task, err := managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(instance.Namespace).Up(ctx, instance.Name, &managementv1.DevPodWorkspaceInstanceUp{
		Spec: managementv1.DevPodWorkspaceInstanceUpSpec{
			Debug:   opt.Debug,
			Options: string(rawOptions),
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating up: %w", err)
	} else if task.Status.TaskID == "" {
		return nil, fmt.Errorf("no up task id returned from server")
	}

	return waitTaskDone(ctx, managementClient, instance, task.Status.TaskID, c.log)
}

func waitTaskDone(ctx context.Context, managementClient kube.Interface, instance *managementv1.DevPodWorkspaceInstance, taskID string, log log.Logger) (*config.Result, error) {
	exitCode, err := observeTask(ctx, managementClient, instance, taskID, log)
	if err != nil {
		return nil, fmt.Errorf("up: %w", err)
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
			TaskID: taskID,
		}, builders.ParameterCodec).
		Do(ctx).
		Into(tasks)
	if err != nil {
		return nil, fmt.Errorf("error getting up result: %w", err)
	} else if len(tasks.Tasks) == 0 || tasks.Tasks[0].Result == nil {
		return nil, fmt.Errorf("up result not found")
	} else if len(tasks.Tasks) > 1 {
		return nil, fmt.Errorf("multiple up results found")
	}

	// unmarshal result
	result := &config.Result{}
	err = json.Unmarshal(tasks.Tasks[0].Result, result)
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
		Target     storagev1.WorkspaceTarget
		Template   *storagev1.TemplateRef
		Parameters string
	}{
		Target:     instance.Spec.Target,
		Template:   instance.Spec.TemplateRef,
		Parameters: instance.Spec.Parameters,
	})
	log.Debug("Starting pro workspace with configuration", string(workspaceConfig))
}

func observeTask(ctx context.Context, managementClient kube.Interface, instance *managementv1.DevPodWorkspaceInstance, taskID string, log log.Logger) (int, error) {
	var (
		exitCode int
		err      error
	)
	errChan := make(chan error, 1)

	printCtx, cancelPrintCtx := context.WithCancel(context.Background())
	defer cancelPrintCtx()

	go func() {
		// cancel ongoing task if context is done
		select {
		case <-ctx.Done():
			timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			defer cancelPrintCtx()

			_, err := managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(instance.Namespace).Cancel(timeoutCtx, instance.Name, &managementv1.DevPodWorkspaceInstanceCancel{
				TaskID: taskID,
			}, metav1.CreateOptions{})
			if err != nil {
				errChan <- err
			} else {
				errChan <- errors.New("canceled")
			}
		case <-errChan:
		case <-printCtx.Done():
		}
	}()
	go func() {
		exitCode, err = printLogs(printCtx, managementClient, instance, taskID, log)
		errChan <- err
	}()

	return exitCode, <-errChan
}

type MessageType byte

const (
	StdoutData MessageType = 0
	StderrData MessageType = 2
	ExitCode   MessageType = 6
)

type Message struct {
	Type     MessageType `json:"type"`
	ExitCode int         `json:"exitCode,omitempty"`
	Bytes    []byte      `json:"bytes,omitempty"`
}

func printLogs(ctx context.Context, managementClient kube.Interface, workspace *managementv1.DevPodWorkspaceInstance, taskID string, logger log.Logger) (int, error) {
	// get logs reader
	logger.Debugf("printing logs of task: %s", taskID)
	logsReader, err := managementClient.Loft().ManagementV1().RESTClient().Get().
		Namespace(workspace.Namespace).
		Resource("devpodworkspaceinstances").
		Name(workspace.Name).
		SubResource("log").
		VersionedParams(&managementv1.DevPodWorkspaceInstanceLogOptions{
			TaskID: taskID,
			Follow: true,
		}, builders.ParameterCodec).
		Stream(ctx)
	if err != nil {
		return -1, fmt.Errorf("error getting task logs: %w", err)
	}
	defer logsReader.Close()

	// create scanner from logs reader
	scanner := bufio.NewScanner(logsReader)

	// Increase the maximum token size to handle very long lines.
	// Here, we set a maximum capacity of 1MB.
	const maxCapacity = 1024 * 1024 // 1MB
	buf := make([]byte, 1024)       // starting buffer size of 1KB
	scanner.Buffer(buf, maxCapacity)

	// create json streamer
	stdoutStreamer, stdoutDone := devpodlog.PipeJSONStream(logger)
	stderrStreamer, stderrDone := devpodlog.PipeJSONStream(logger.ErrorStreamOnly())
	defer func() {
		// close the streams
		stdoutStreamer.Close()
		stderrStreamer.Close()

		// wait for the streams to be closed
		<-stdoutDone
		<-stderrDone
	}()

	// loop over all lines
	for scanner.Scan() {
		line := scanner.Text()

		// parse message
		message := &Message{}
		if err := json.Unmarshal([]byte(line), message); err != nil {
			return -1, fmt.Errorf("error parsing JSON from logs reader: %w, line: %s", err, line)
		}

		// write message to stdout or stderr
		if message.Type == StdoutData {
			if _, err := stdoutStreamer.Write(message.Bytes); err != nil {
				logger.Debugf("error read stdout: %v", err)
				return 1, err
			}
		} else if message.Type == StderrData {
			if _, err := stderrStreamer.Write(message.Bytes); err != nil {
				logger.Debugf("error read stderr: %v", err)
				return 1, err
			}
		} else if message.Type == ExitCode {
			logger.Debugf("exit code: %d", message.ExitCode)
			return message.ExitCode, nil
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return 0, nil
		}
		return -1, fmt.Errorf("logs reader error: %w", err)
	}

	return 0, nil
}

const (
	TaskStatusRunning = "Running"
	TaskStatusSucceed = "Succeeded"
	TaskStatusFailed  = "Failed"
)
const (
	TaskTypeUp     = "up"
	TaskTypeStop   = "stop"
	TaskTypeDelete = "delete"
)

func findActiveUpTask(ctx context.Context, managementClient kube.Interface, instance *managementv1.DevPodWorkspaceInstance) (*managementv1.DevPodWorkspaceInstanceTask, error) {
	tasks := &managementv1.DevPodWorkspaceInstanceTasks{}
	err := managementClient.Loft().ManagementV1().RESTClient().Get().
		Namespace(instance.Namespace).
		Resource("devpodworkspaceinstances").
		Name(instance.Name).
		SubResource("tasks").
		Do(ctx).
		Into(tasks)

	for _, task := range tasks.Tasks {
		if task.Status == TaskStatusRunning && task.Type == TaskTypeUp {
			return &task, nil
		}
	}

	return nil, err
}
