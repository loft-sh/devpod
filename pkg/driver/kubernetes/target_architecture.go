package kubernetes

import (
	"context"
	"fmt"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *KubernetesDriver) TargetArchitecture(ctx context.Context, workspaceId string) (string, error) {
	if k.options.Architecture != "" {
		return k.options.Architecture, nil
	}

	k.Log.Debugf("Getting target architecture for workspace '%s'", workspaceId)
	defer k.Log.Debugf("Done getting target architecture for workspace '%s'", workspaceId)

	// get all nodes
	nodes, err := k.client.Client().CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		if kerrors.IsForbidden(err) {
			return "", fmt.Errorf("you don't have permission to list nodes in the Kubernetes cluster, please set the cluster architecture manually via provider options")
		}

		return "", fmt.Errorf("list nodes: %w", err)
	}

	// check if there are mixed architectures
	architecture := ""
	for _, node := range nodes.Items {
		if architecture == "" {
			architecture = node.Status.NodeInfo.Architecture
		} else if architecture != node.Status.NodeInfo.Architecture {
			return "", fmt.Errorf("mixed architectures in the Kubernetes cluster, please set the cluster architecture manually via provider options")
		}
	}

	return architecture, nil
}
