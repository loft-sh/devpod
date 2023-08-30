package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/loft-sh/devpod/pkg/driver"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *kubernetesDriver) createPersistentVolumeClaim(
	ctx context.Context,
	id string,
	options *driver.RunOptions,
) error {
	pvcString, err := k.buildPersistentVolumeClaim(id, options)
	if err != nil {
		return err
	}

	k.Log.Infof("Create Persistent Volume Claim '%s'", id)
	buf := &bytes.Buffer{}
	err = k.runCommand(ctx, []string{"create", "-f", "-"}, strings.NewReader(pvcString), buf, buf)
	if err != nil {
		return errors.Wrapf(err, "create pvc: %s", buf.String())
	}

	return nil
}

func (k *kubernetesDriver) buildPersistentVolumeClaim(
	id string,
	options *driver.RunOptions,
) (string, error) {
	containerInfo, err := k.getDevContainerInformation(id, options)
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
			Name:   id,
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
	id string,
	options *driver.RunOptions,
) (string, error) {
	containerInfo, err := json.Marshal(&DevContainerInfo{
		WorkspaceID: id,
		Options:     options,
	})
	if err != nil {
		return "", err
	}

	return string(containerInfo), nil
}
