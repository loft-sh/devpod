package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

func (k *kubernetesDriver) createPersistentVolumeClaim(
	ctx context.Context,
	name string,
	parsedConfig *config.DevContainerConfig,
	mergedConfig *config.MergedDevContainerConfig,
	imageName string,
	workspaceMount string,
	labels []string,
	imageDetails *config.ImageDetails,
) error {
	pvcString, err := k.buildPersistentVolumeClaim(name, parsedConfig, mergedConfig, imageName, workspaceMount, labels, imageDetails)
	if err != nil {
		return err
	}

	k.Log.Infof("Create Persistent Volume '%s'", name)
	buf := &bytes.Buffer{}
	err = k.runCommand(ctx, []string{"create", "-f", "-"}, strings.NewReader(pvcString), buf, buf)
	if err != nil {
		return errors.Wrapf(err, "create pvc: %s", buf.String())
	}

	return nil
}

func (k *kubernetesDriver) buildPersistentVolumeClaim(
	name string,
	parsedConfig *config.DevContainerConfig,
	mergedConfig *config.MergedDevContainerConfig,
	imageName string,
	workspaceMount string,
	labels []string,
	imageDetails *config.ImageDetails,
) (string, error) {
	containerInfo, err := k.getDevContainerInformation(parsedConfig, mergedConfig, imageName, workspaceMount, labels, imageDetails)
	if err != nil {
		return "", err
	}

	size := "10Gi"
	if k.config.PersistentVolumeSize != "" {
		size = k.config.PersistentVolumeSize + "Gi"
	}
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return "", errors.Wrap(err, "parse persistent volume size")
	}

	raw, err := json.Marshal(&corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"devpod": "true",
			},
			Annotations: map[string]string{
				DevContainerInfoAnnotation: containerInfo,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: quantity,
				},
			},
		},
	})
	if err != nil {
		return "", err
	}

	return string(raw), nil
}

func (k *kubernetesDriver) getDevContainerInformation(
	parsedConfig *config.DevContainerConfig,
	mergedConfig *config.MergedDevContainerConfig,
	imageName string,
	workspaceMount string,
	labels []string,
	imageDetails *config.ImageDetails,
) (string, error) {
	containerInfo, err := json.Marshal(&DevContainerInfo{
		MergedConfig:   mergedConfig,
		ParsedConfig:   parsedConfig,
		ImageDetails:   imageDetails,
		ImageName:      imageName,
		WorkspaceMount: workspaceMount,
		Labels:         labels,
	})
	if err != nil {
		return "", err
	}

	return string(containerInfo), nil
}

func (k *kubernetesDriver) getID(labels []string) (string, error) {
	id := config.ListToObject(labels)[config.DockerIDLabel]
	if id == "" {
		return "", fmt.Errorf("id label is missing")
	}

	return "devpod-" + id, nil
}
