package kubernetes

import corev1 "k8s.io/api/core/v1"

func IsReady(status *corev1.ContainerStatus) bool {
	return status.Ready
}

func IsWaiting(status *corev1.ContainerStatus) bool {
	return status.State.Waiting != nil
}

func IsTerminated(status *corev1.ContainerStatus) bool {
	return status.State.Terminated != nil
}

func IsStarted(status *corev1.ContainerStatus) bool {
	return status.Started != nil && *status.Started
}

func IsRunning(status *corev1.ContainerStatus) bool {
	return status.State.Running != nil
}

func Succeeded(status *corev1.ContainerStatus) bool {
	return status.State.Terminated != nil && status.State.Terminated.ExitCode == 0
}

func IsCritical(status *corev1.ContainerStatus) bool {
	return criticalStatus[status.State.Waiting.Reason]
}

var criticalStatus = map[string]bool{
	"Error":                      true,
	"Unknown":                    true,
	"ImagePullBackOff":           true,
	"CrashLoopBackOff":           true,
	"RunContainerError":          true,
	"ErrImagePull":               true,
	"CreateContainerConfigError": true,
	"InvalidImageName":           true,
}
