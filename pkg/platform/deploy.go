package platform

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/loft-sh/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
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

func WaitForPodReady(ctx context.Context, kubeClient kubernetes.Interface, namespace string, log log.Logger) (*corev1.Pod, error) {
	// wait until we have a running loft pod
	now := time.Now()
	pod := &corev1.Pod{}
	err := wait.PollUntilContextTimeout(ctx, time.Second*2, Timeout(), true, func(ctx context.Context) (bool, error) {
		pods, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=loft",
		})
		if err != nil {
			log.Warnf("Error trying to retrieve %s pod: %v", "DevPod Pro", err)
			return false, nil
		} else if len(pods.Items) == 0 {
			if time.Now().After(now.Add(time.Second * 10)) {
				log.Infof("Still waiting for a %s pod...", "DevPod Pro")
				now = time.Now()
			}
			return false, nil
		}

		sort.Slice(pods.Items, func(i, j int) bool {
			return pods.Items[i].CreationTimestamp.After(pods.Items[j].CreationTimestamp.Time)
		})

		loftPod := &pods.Items[0]
		found := false
		for _, containerStatus := range loftPod.Status.ContainerStatuses {
			if containerStatus.State.Running != nil && containerStatus.Ready {
				if containerStatus.Name == "manager" {
					found = true
				}

				continue
			} else if containerStatus.State.Terminated != nil || (containerStatus.State.Waiting != nil && CriticalStatus[containerStatus.State.Waiting.Reason]) {
				reason := ""
				message := ""
				if containerStatus.State.Terminated != nil {
					reason = containerStatus.State.Terminated.Reason
					message = containerStatus.State.Terminated.Message
				} else if containerStatus.State.Waiting != nil {
					reason = containerStatus.State.Waiting.Reason
					message = containerStatus.State.Waiting.Message
				}

				out, err := kubeClient.CoreV1().Pods(namespace).GetLogs(loftPod.Name, &corev1.PodLogOptions{
					Container: "manager",
				}).Do(context.Background()).Raw()
				if err != nil {
					return false, fmt.Errorf("there seems to be an issue with %s starting up: %s (%s). Please reach out to our support at https://loft.sh/", "DevPod Pro", message, reason)
				}
				if strings.Contains(string(out), "register instance: Post \"https://license.loft.sh/register\": dial tcp") {
					return false, fmt.Errorf("%[1]s logs: \n%[2]v \nThere seems to be an issue with %[1]s starting up. Looks like you try to install %[1]s into an air-gapped environment, please reach out to our support at https://loft.sh/ for an offline license", "DevPod Pro", string(out))
				}

				return false, fmt.Errorf("%[1]s logs: \n%v \nThere seems to be an issue with %[1]s starting up: %[2]s (%[3]s). Please reach out to our support at https://loft.sh/", "DevPod Pro", string(out), message, reason)
			} else if containerStatus.State.Waiting != nil && time.Now().After(now.Add(time.Second*10)) {
				if containerStatus.State.Waiting.Message != "" {
					log.Infof("Please keep waiting, %s container is still starting up: %s (%s)", "DevPod Pro", containerStatus.State.Waiting.Message, containerStatus.State.Waiting.Reason)
				} else if containerStatus.State.Waiting.Reason != "" {
					log.Infof("Please keep waiting, %s container is still starting up: %s", "DevPod Pro", containerStatus.State.Waiting.Reason)
				} else {
					log.Infof("Please keep waiting, %s container is still starting up...", "DevPod Pro")
				}

				now = time.Now()
			}

			return false, nil
		}

		pod = loftPod
		return found, nil
	})
	if err != nil {
		return nil, err
	}

	return pod, nil
}
