package kubernetes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	perrors "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// CriticalStatus container status
var CriticalStatus = map[string]bool{
	"Error":                      true,
	"Unknown":                    true,
	"ImagePullBackOff":           true,
	"CrashLoopBackOff":           true,
	"RunContainerError":          true,
	"ErrImagePull":               true,
	"CreateContainerConfigError": true,
	"InvalidImageName":           true,
}

func (k *kubernetesDriver) waitPodRunning(ctx context.Context, id string) (*corev1.Pod, error) {
	nextMessage := time.Now().Add(time.Second * 5)

	var pod *corev1.Pod
	err := wait.PollImmediate(time.Second, time.Minute*5, func() (bool, error) {
		var err error
		pod, err = k.getPod(ctx, id)
		now := time.Now()
		if err != nil {
			return false, err
		} else if pod == nil {
			return true, nil
		}

		// check pod for problems
		if pod.DeletionTimestamp != nil {
			if now.After(nextMessage) {
				k.Log.Infof("Waiting, since pod '%s' is terminating", id)
				nextMessage = now.Add(time.Second * 5)
			}
			return false, nil
		}

		// check pod status
		if len(pod.Status.ContainerStatuses) < len(pod.Spec.Containers) {
			if now.After(nextMessage) {
				k.Log.Infof("Waiting, since pod '%s' is waiting to start", id)
				nextMessage = now.Add(time.Second * 5)
			}
			return false, nil
		}

		// check container status
		for _, c := range pod.Status.InitContainerStatuses {
			// is waiting
			if c.State.Waiting != nil {
				if CriticalStatus[c.State.Waiting.Reason] {
					return false, fmt.Errorf("pod '%s' init container '%s' is waiting to start: %s (%s)", id, c.Name, c.State.Waiting.Message, c.State.Waiting.Reason)
				}

				if now.After(nextMessage) {
					k.Log.Infof("Waiting, since pod '%s' is waiting to start: %s (%s)", id, c.State.Waiting.Message, c.State.Waiting.Reason)
					nextMessage = now.Add(time.Second * 5)
				}
				return false, nil
			}

			// is terminated
			if c.State.Terminated != nil && c.State.Terminated.ExitCode != 0 {
				return false, fmt.Errorf("pod '%s' init container '%s' is terminated: %s (%s)", id, c.Name, c.State.Terminated.Message, c.State.Terminated.Reason)
			}

			// is running
			if c.State.Running != nil {
				if now.After(nextMessage) {
					k.Log.Infof("Waiting, since pod '%s' init container '%s' is running", id, c.Name)
					nextMessage = now.Add(time.Second * 5)
				}
				return false, nil
			}
		}

		// check container status
		for _, c := range pod.Status.ContainerStatuses {
			// delete succeeded pods
			if c.State.Terminated != nil && c.State.Terminated.ExitCode == 0 {
				// delete pod that is succeeded
				k.Log.Debugf("Delete Pod '%s' because it is succeeded", id)
				err = k.deletePod(ctx, id)
				if err != nil {
					return false, err
				}

				return false, nil
			}

			// is waiting
			if c.State.Waiting != nil {
				if CriticalStatus[c.State.Waiting.Reason] {
					return false, fmt.Errorf("pod '%s' container '%s' is waiting to start: %s (%s)", id, c.Name, c.State.Waiting.Message, c.State.Waiting.Reason)
				}

				if now.After(nextMessage) {
					k.Log.Infof("Waiting, since pod '%s' is waiting to start: %s (%s)", id, c.State.Waiting.Message, c.State.Waiting.Reason)
					nextMessage = now.Add(time.Second * 5)
				}
				return false, nil
			}

			// is terminated
			if c.State.Terminated != nil {
				return false, fmt.Errorf("pod '%s' container '%s' is terminated: %s (%s)", id, c.Name, c.State.Terminated.Message, c.State.Terminated.Reason)
			}

			// is not ready
			if !c.Ready {
				if now.After(nextMessage) {
					k.Log.Infof("Waiting, since pod '%s' container '%s' is not ready yet", id, c.Name)
					nextMessage = now.Add(time.Second * 5)
				}
				return false, nil
			}
		}

		return true, nil
	})
	return pod, err
}

func (k *kubernetesDriver) getPod(ctx context.Context, id string) (*corev1.Pod, error) {
	// try to find pod
	out, err := k.buildCmd(ctx, []string{"get", "pod", id, "--ignore-not-found", "-o", "json"}).Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return nil, fmt.Errorf("find container: %s", strings.TrimSpace(string(exitError.Stderr)))
		}

		return nil, perrors.Wrap(err, "find container")
	} else if len(out) == 0 {
		return nil, nil
	}

	// try to unmarshal pod
	pod := &corev1.Pod{}
	err = json.Unmarshal(out, pod)
	if err != nil {
		return nil, perrors.Wrap(err, "unmarshal pod")
	}

	return pod, nil
}
