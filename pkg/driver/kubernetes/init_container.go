package kubernetes

import (
	"fmt"
	"strings"

	"github.com/loft-sh/devpod/pkg/driver"
	corev1 "k8s.io/api/core/v1"
)

func (k *KubernetesDriver) getInitContainers(options *driver.RunOptions, pod *corev1.Pod, initialize bool) ([]corev1.Container, error) {
	if !initialize {
		retContainers := []corev1.Container{}
		// don't build init container and clean up existing one if defined
		for _, container := range pod.Spec.InitContainers {
			if container.Name == InitContainerName {
				continue
			}
			retContainers = append(retContainers, container)
		}

		return retContainers, nil
	}

	commands := []string{}
	// find the volume type mounts
	volumeMounts := []corev1.VolumeMount{}
	for idx, mount := range options.Mounts {
		if mount.Type != "volume" {
			continue
		}

		volumeMount := getVolumeMount(idx+1, mount)
		copyFrom := volumeMount.MountPath
		volumeMount.MountPath = "/" + volumeMount.SubPath
		volumeMounts = append(volumeMounts, volumeMount)
		commands = append(commands, fmt.Sprintf(`cp -a %s/. %s/ || true`, strings.TrimRight(copyFrom, "/"), strings.TrimRight(volumeMount.MountPath, "/")))
	}

	retContainers := []corev1.Container{}

	// merge with existing init container if it exists
	var existingInitContainer *corev1.Container
	for i, container := range pod.Spec.InitContainers {
		if container.Name == InitContainerName {
			existingInitContainer = &pod.Spec.InitContainers[i]
		} else {
			retContainers = append(retContainers, container)
		}
	}

	// check if there is at least one mount
	if len(volumeMounts) == 0 {
		return retContainers, nil
	}

	securityContext := &corev1.SecurityContext{
		RunAsUser:    &[]int64{0}[0],
		RunAsGroup:   &[]int64{0}[0],
		RunAsNonRoot: &[]bool{false}[0],
	}
	if k.options.StrictSecurity == "true" {
		securityContext = nil
	}

	resources := corev1.ResourceRequirements{}
	if existingInitContainer != nil {
		resources = existingInitContainer.Resources
	}

	initContainer := corev1.Container{
		Name:            InitContainerName,
		Image:           options.Image,
		Command:         []string{"sh"},
		Args:            []string{"-c", strings.Join(commands, "\n") + "\n"},
		Resources:       resources,
		VolumeMounts:    volumeMounts,
		SecurityContext: securityContext,
	}

	if existingInitContainer != nil {
		initContainer.Env = append(existingInitContainer.Env, initContainer.Env...)
		initContainer.EnvFrom = existingInitContainer.EnvFrom
		initContainer.Ports = existingInitContainer.Ports
		initContainer.VolumeMounts = append(existingInitContainer.VolumeMounts, initContainer.VolumeMounts...)
		initContainer.ImagePullPolicy = existingInitContainer.ImagePullPolicy

		if initContainer.SecurityContext == nil && existingInitContainer.SecurityContext != nil {
			initContainer.SecurityContext = existingInitContainer.SecurityContext
		}
	}

	retContainers = append(retContainers, initContainer)
	return retContainers, nil
}
