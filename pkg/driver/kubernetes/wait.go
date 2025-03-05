package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/driver/kubernetes/throttledlogger"
	perrors "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (k *KubernetesDriver) waitPodRunning(ctx context.Context, id string) (*corev1.Pod, error) {
	throttledLogger := throttledlogger.NewThrottledLogger(k.Log, time.Second*5)

	timeoutDuration, err := time.ParseDuration(k.options.PodTimeout)
	if err != nil {
		return nil, perrors.Wrap(err, "parse pod timeout")
	}

	started := time.Now()
	var pod *corev1.Pod
	err = wait.PollUntilContextTimeout(ctx, time.Second, timeoutDuration, true, func(ctx context.Context) (bool, error) {
		var err error
		pod, err = k.getPod(ctx, id)
		if err != nil {
			return false, err
		} else if pod == nil {
			return true, nil
		}

		// check pod for problems
		if pod.DeletionTimestamp != nil {
			throttledLogger.Infof("Waiting, since pod '%s' is terminating", id)
			return false, nil
		}

		// Let's print all conditions that are false to help people troubleshoot infra issues
		condMsg := ""
		if time.Since(started) > 45*time.Second { // start printing conditions after a delay
			for _, cond := range pod.Status.Conditions {
				if cond.Status == corev1.ConditionFalse {
					condMsg += fmt.Sprintf("Condition \"%s\" is %s\n", cond.Type, cond.Status)
					if cond.Reason != "" {
						condMsg += fmt.Sprintf("%s Reason: %s\n", cond.Type, cond.Reason)
					}
					if cond.Message != "" {
						condMsg += fmt.Sprintf("%s Message: %s\n", cond.Type, cond.Message)
					}
				}
			}
		}

		// check pod status
		if len(pod.Status.ContainerStatuses) < len(pod.Spec.Containers) {
			msg := fmt.Sprintf("Waiting, since pod '%s' is starting", id)
			if condMsg != "" {
				msg += fmt.Sprintf("\n%s", strings.TrimSpace(condMsg))
			}
			throttledLogger.Infof("%s", msg)
			return false, nil
		}

		// check init container status
		for _, c := range pod.Status.InitContainerStatuses {
			containerStatus := &c
			if IsWaiting(containerStatus) {
				if IsCritical(containerStatus) {
					return false, fmt.Errorf("pod '%s' init container '%s' is waiting to start: %s (%s)", id, c.Name, c.State.Waiting.Message, c.State.Waiting.Reason)
				}
				if c.State.Waiting.Message == "" {
					throttledLogger.Infof("Waiting, since pod '%s' init container '%s' is waiting to start: %s", id, c.Name, c.State.Waiting.Reason)
				} else {
					throttledLogger.Infof("Waiting, since pod '%s' init container '%s' is waiting to start: %s (%s)", id, c.Name, c.State.Waiting.Message, c.State.Waiting.Reason)
				}

				return false, nil
			}

			if IsTerminated(containerStatus) && !Succeeded(containerStatus) {
				return false, fmt.Errorf("pod '%s' init container '%s' is terminated: %s (%s)", id, c.Name, c.State.Terminated.Message, c.State.Terminated.Reason)
			}

			container, err := getContainer(pod.Spec.InitContainers, c.Name)
			if err != nil {
				throttledLogger.Infof("Could not find container '%s'", c.Name)
				return false, err
			}

			restartable := restartableInitContainer(container.RestartPolicy)
			if restartable {
				if !IsStarted(containerStatus) || !IsReady(containerStatus) {
					throttledLogger.Infof("Waiting, since pod '%s' init container '%s' is not ready yet", id, c.Name)
					return false, nil
				}
			} else {
				if IsRunning(containerStatus) {
					throttledLogger.Infof("Waiting, since pod '%s' init container '%s' is running", id, c.Name)
					return false, nil
				}
			}
		}

		// check container status
		for _, c := range pod.Status.ContainerStatuses {
			containerStatus := &c
			// delete succeeded pods
			if IsTerminated(containerStatus) && Succeeded(containerStatus) {
				// delete pod that is succeeded
				k.Log.Debugf("Delete Pod '%s' because it is succeeded", id)
				err = k.waitPodDeleted(ctx, id)
				if err != nil {
					return false, err
				}

				return false, nil
			}

			if IsWaiting(containerStatus) {
				if IsCritical(containerStatus) {
					return false, fmt.Errorf("pod '%s' container '%s' is waiting to start: %s (%s)", id, c.Name, c.State.Waiting.Message, c.State.Waiting.Reason)
				}
				if c.State.Waiting.Message == "" {
					throttledLogger.Infof("Waiting, since pod '%s' container '%s' is waiting to start: %s", id, c.Name, c.State.Waiting.Reason)
				} else {
					throttledLogger.Infof("Waiting, since pod '%s' container '%s' is waiting to start: %s (%s)", id, c.Name, c.State.Waiting.Message, c.State.Waiting.Reason)
				}

				return false, nil
			}

			if IsTerminated(containerStatus) {
				return false, fmt.Errorf("pod '%s' container '%s' is terminated: %s (%s)", id, c.Name, c.State.Terminated.Message, c.State.Terminated.Reason)
			}

			if !IsReady(containerStatus) {
				throttledLogger.Infof("Waiting, since pod '%s' container '%s' is not ready yet", id, c.Name)
				return false, nil
			}
		}

		return true, nil
	})

	return pod, err
}

func (k *KubernetesDriver) getPod(ctx context.Context, id string) (*corev1.Pod, error) {
	// try to find pod
	pod, err := k.client.Client().CoreV1().Pods(k.namespace).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("find container: %w", err)
	}

	return pod, nil
}

func getContainer(containers []corev1.Container, name string) (*corev1.Container, error) {
	for _, c := range containers {
		if c.Name == name {
			return &c, nil
		}
	}

	return nil, fmt.Errorf("cannot find pod container with name %s", name)
}

func restartableInitContainer(p *corev1.ContainerRestartPolicy) bool {
	return p != nil && *p == corev1.ContainerRestartPolicyAlways
}

func (k *KubernetesDriver) waitPodDeleted(ctx context.Context, id string) error {
	err := k.client.Client().CoreV1().Pods(k.namespace).Delete(ctx, id, metav1.DeleteOptions{
		GracePeriodSeconds: &[]int64{10}[0],
	})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("delete pod: %w", err)
	}

	err = wait.PollUntilContextTimeout(ctx, time.Second, time.Minute*2, true, func(ctx context.Context) (bool, error) {
		_, err := k.client.Client().CoreV1().Pods(k.namespace).Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return errors.New("timeout waiting for pod to be deleted")
	}

	return nil
}
