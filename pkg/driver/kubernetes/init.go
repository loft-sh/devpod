package kubernetes

import (
	"fmt"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

func (k *kubernetesDriver) getInitContainer(
	mergedConfig *config.MergedDevContainerConfig,
	imageName string,
) ([]corev1.Container, error) {
	commands := []string{}

	// find the volume type mounts
	volumeMounts := []corev1.VolumeMount{}
	for idx, mount := range mergedConfig.Mounts {
		if mount.Type != "volume" {
			continue
		}

		volumeMount := getVolumeMount(idx+1, mount)
		copyFrom := volumeMount.MountPath
		volumeMount.MountPath = "/" + volumeMount.SubPath
		volumeMounts = append(volumeMounts, volumeMount)
		commands = append(commands, fmt.Sprintf(`cp -a %s/. %s/`, strings.TrimRight(copyFrom, "/"), strings.TrimRight(volumeMount.MountPath, "/")))
	}

	// check if there is at least one mount
	if len(volumeMounts) == 0 {
		return nil, nil
	}

	// return container
	return []corev1.Container{
		{
			Name:         "devpod-init",
			Image:        imageName,
			Command:      []string{"/bin/sh"},
			Args:         []string{"-c", strings.Join(commands, "\n") + "\n"},
			VolumeMounts: volumeMounts,
			SecurityContext: &corev1.SecurityContext{
				RunAsUser:    &[]int64{0}[0],
				RunAsGroup:   &[]int64{0}[0],
				RunAsNonRoot: &[]bool{false}[0],
			},
		},
	}, nil
}
