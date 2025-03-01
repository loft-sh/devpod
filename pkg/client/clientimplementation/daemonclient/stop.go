package daemonclient

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/apiserver/pkg/builders"
	clientpkg "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/kube"
	"github.com/loft-sh/log"

	_ "github.com/loft-sh/api/v4/pkg/apis/management/install" // Install the management group to ensure the option types are registered
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

	_, err = printLogs(ctx, managementClient, workspace, retStop.Status.TaskID, os.Stdout, os.Stderr, c.log)
	if err != nil {
		return fmt.Errorf("error getting stop logs: %w", err)
	}

	return nil
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

func printLogs(ctx context.Context, managementClient kube.Interface, workspace *managementv1.DevPodWorkspaceInstance, taskID string, stdout io.Writer, stderr io.Writer, log log.Logger) (int, error) {
	// get logs reader
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

	// loop over all lines
	for scanner.Scan() {
		line := scanner.Text()

		// parse message
		message := &Message{}
		if err := json.Unmarshal([]byte(line), message); err != nil {
			return -1, fmt.Errorf("error parsing JSON from logs reader: %w, line: %s", err, string(line))
		}

		// write message to stdout or stderr
		if message.Type == StdoutData {
			if _, err := stdout.Write(message.Bytes); err != nil {
				log.Debugf("error read stdout: %v", err)
				return 1, err
			}
		} else if message.Type == StderrData {
			if _, err := stderr.Write(message.Bytes); err != nil {
				log.Debugf("error read stderr: %v", err)
				return 1, err
			}
		} else if message.Type == ExitCode {
			log.Debugf("exit code: %d", message.ExitCode)
			return int(message.ExitCode), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return -1, fmt.Errorf("logs reader error: %w", err)
	}

	return 0, nil
}
