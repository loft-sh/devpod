package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	k.Log.Infof("Create Persistent Volume Claim '%s'", name)
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
		size = k.config.PersistentVolumeSize
	}
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return "", errors.Wrapf(err, "parse persistent volume size '%s'", size)
	}

	var storageClassName *string
	if k.config.StorageClassName != "" {
		storageClassName = &k.config.StorageClassName
	}
	accessMode := []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	if k.config.PVCAccessMode != "" {
		switch k.config.PVCAccessMode {
		case "RWO":
			accessMode = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		case "ROX":
			accessMode = []corev1.PersistentVolumeAccessMode{corev1.ReadOnlyMany}
		case "RWX":
			accessMode = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
		case "RWOP":
			accessMode = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOncePod}
		default:
			accessMode = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		}
	}

	pvc := &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: DevPodLabels,
			Annotations: map[string]string{
				DevContainerInfoAnnotation: containerInfo,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessMode,
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: quantity,
				},
			},
			StorageClassName: storageClassName,
		},
	}

	raw, err := json.Marshal(pvc)
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
