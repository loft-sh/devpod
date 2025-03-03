package kubernetes

import (
	"context"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	corev1 "k8s.io/api/core/v1"
)

func (k *KubernetesDriver) FindDevContainer(ctx context.Context, workspaceId string) (*config.ContainerDetails, error) {
	k.Log.Debugf("Finding devcontainer for workspace '%s'", workspaceId)
	defer k.Log.Debugf("Done finding devcontainer for workspace '%s'", workspaceId)

	workspaceId = getID(workspaceId)

	pvc, containerInfo, err := k.getDevContainerPvc(ctx, workspaceId)
	if err != nil {
		return nil, err
	} else if pvc == nil {
		return nil, nil
	}

	// check pod
	pod, err := k.getPod(ctx, pvc.Name)
	if err != nil {
		k.Log.Infof("Error finding pod: %v", err)
		k.Log.Warn("If the pod does not come up automatically it is stuck in an error state. Recreate the workspace to recover from this")
		pod = nil
	}

	// determine status
	status := "exited"
	if pod != nil && isPodRunning(pod) {
		status = "running"
	}

	// check started
	startedAt := pvc.CreationTimestamp.String()
	if pod != nil {
		startedAt = pod.CreationTimestamp.String()
	}

	return &config.ContainerDetails{
		ID:      pvc.Name,
		Created: pvc.CreationTimestamp.String(),
		State: config.ContainerDetailsState{
			Status:    status,
			StartedAt: startedAt,
		},
		Config: config.ContainerDetailsConfig{
			Labels: config.ListToObject(containerInfo.Options.Labels),
		},
	}, nil
}

func isPodRunning(pod *corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodRunning
}
