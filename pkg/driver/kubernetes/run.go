package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DevContainerInfoAnnotation = "devpod.sh/info"
const DevContainerName = "devpod"

var DevPodLabels = map[string]string{
	"devpod.sh/created": "true",
}

type DevContainerInfo struct {
	WorkspaceID string
	Options     *driver.RunOptions
}

func (k *kubernetesDriver) RunDevContainer(
	ctx context.Context,
	workspaceId string,
	options *driver.RunOptions,
) error {
	workspaceId = getID(workspaceId)

	// namespace
	if k.namespace != "" && k.config.CreateNamespace == "true" {
		k.Log.Debugf("Create namespace '%s'", k.namespace)
		buf := &bytes.Buffer{}
		err := k.runCommand(ctx, []string{"create", "ns", k.namespace}, nil, buf, buf)
		if err != nil {
			k.Log.Debugf("Error creating namespace: %v", err)
		}
	}

	// check if persistent volume claim already exists
	initialize := false
	pvc, _, err := k.getDevContainerPvc(ctx, workspaceId)
	if err != nil {
		return err
	} else if pvc == nil {
		// create persistent volume claim
		err = k.createPersistentVolumeClaim(ctx, workspaceId, options)
		if err != nil {
			return err
		}

		initialize = true
	}

	// create dev container
	err = k.runContainer(ctx, workspaceId, options, initialize)
	if err != nil {
		return err
	}

	return nil
}

func (k *kubernetesDriver) runContainer(
	ctx context.Context,
	id string,
	options *driver.RunOptions,
	initialize bool,
) (err error) {
	// get workspace mount
	mount := options.WorkspaceMount
	if mount.Target == "" {
		return fmt.Errorf("workspace mount target is empty")
	}

	// read pod template
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
	}
	if len(k.config.PodManifestTemplate) > 0 {
		podManifestTemplatePath, err := filepath.Abs(k.config.PodManifestTemplate)
		if err != nil {
			return err
		}
		pod, err = getPodTemplate(podManifestTemplatePath)
		if err != nil {
			return err
		}
	}

	// get init container
	var initContainer []corev1.Container
	if initialize {
		initContainer, err = k.getInitContainer(options)
		if err != nil {
			return errors.Wrap(err, "build init container")
		}
	}

	// loop over volume mounts
	volumeMounts := []corev1.VolumeMount{getVolumeMount(0, mount)}
	for idx, mount := range options.Mounts {
		volumeMount := getVolumeMount(idx+1, mount)
		if mount.Type == "bind" || mount.Type == "volume" {
			volumeMounts = append(volumeMounts, volumeMount)
		} else {
			k.Log.Warnf("Unsupported mount type '%s' in mount '%s', will skip", mount.Type, mount.String())
		}
	}

	// capabilities
	var capabilities *corev1.Capabilities
	if len(options.CapAdd) > 0 {
		capabilities = &corev1.Capabilities{}
		for _, cap := range options.CapAdd {
			capabilities.Add = append(capabilities.Add, corev1.Capability(cap))
		}
	}

	// env vars
	envVars := []corev1.EnvVar{}
	for k, v := range options.Env {
		envVars = append(envVars, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	// service account
	serviceAccount := ""
	if k.config.ServiceAccount != "" {
		serviceAccount = k.config.ServiceAccount

		// create service account
		err = k.createServiceAccount(ctx, id, serviceAccount)
		if err != nil {
			return fmt.Errorf("create service account: %w", err)
		}
	}

	// labels
	labels, err := getLabels(pod, k.config.Labels)
	if err != nil {
		return err
	}

	// node selector
	nodeSelector, err := getNodeSelector(pod, k.config.NodeSelector)
	if err != nil {
		return err
	}

	// parse resources
	resources := parseResources(k.config.Resources, k.Log)

	// create the pod manifest
	pod.ObjectMeta.Name = id
	pod.ObjectMeta.Labels = labels

	pod.Spec.ServiceAccountName = serviceAccount
	pod.Spec.NodeSelector = nodeSelector
	pod.Spec.InitContainers = append(initContainer, pod.Spec.InitContainers...)
	pod.Spec.Containers = getContainers(pod, options.Image, options.Entrypoint, options.Cmd, envVars, volumeMounts, capabilities, resources, options.Privileged)
	pod.Spec.Volumes = getVolumes(pod, id)
	pod.Spec.RestartPolicy = corev1.RestartPolicyNever

	// marshal the pod
	podRaw, err := json.Marshal(pod)
	if err != nil {
		return err
	}
	k.Log.Debugf("Create pod with: %s", string(podRaw))

	// create the pod
	k.Log.Infof("Create Pod '%s'", id)
	buf := &bytes.Buffer{}
	err = k.runCommand(ctx, []string{"create", "-f", "-"}, strings.NewReader(string(podRaw)), buf, buf)
	if err != nil {
		return errors.Wrapf(err, "create pod: %s", buf.String())
	}

	// wait for pod running
	k.Log.Infof("Waiting for DevContainer Pod '%s' to come up...", id)
	_, err = k.waitPodRunning(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

func getContainers(
	pod *corev1.Pod,
	imageName,
	entrypoint string,
	args []string,
	envVars []corev1.EnvVar,
	volumeMounts []corev1.VolumeMount,
	capabilities *corev1.Capabilities,
	resources corev1.ResourceRequirements,
	privileged *bool,
) []corev1.Container {
	devPodContainer := corev1.Container{
		Name:         DevContainerName,
		Image:        imageName,
		Command:      []string{entrypoint},
		Args:         args,
		Env:          envVars,
		Resources:    resources,
		VolumeMounts: volumeMounts,
		SecurityContext: &corev1.SecurityContext{
			Capabilities: capabilities,
			Privileged:   privileged,
			RunAsUser:    &[]int64{0}[0],
			RunAsGroup:   &[]int64{0}[0],
			RunAsNonRoot: &[]bool{false}[0],
		},
	}

	// merge with existing container if it exists
	var existingDevPodContainer *corev1.Container
	retContainers := []corev1.Container{}
	if pod != nil {
		for i, container := range pod.Spec.Containers {
			if container.Name == DevContainerName {
				existingDevPodContainer = &pod.Spec.Containers[i]
			} else {
				retContainers = append(retContainers, container)
			}
		}
	}
	if existingDevPodContainer != nil {
		devPodContainer.Env = append(existingDevPodContainer.Env, devPodContainer.Env...)
		devPodContainer.EnvFrom = existingDevPodContainer.EnvFrom
		devPodContainer.Ports = existingDevPodContainer.Ports
		devPodContainer.VolumeMounts = append(existingDevPodContainer.VolumeMounts, devPodContainer.VolumeMounts...)
	}
	retContainers = append(retContainers, devPodContainer)

	return retContainers
}

func getVolumes(pod *corev1.Pod, id string) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: "devpod",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: id,
				},
			},
		},
	}

	if pod.Spec.Volumes != nil {
		volumes = append(volumes, pod.Spec.Volumes...)
	}

	return volumes
}

func getVolumeMount(idx int, mount *config.Mount) corev1.VolumeMount {
	subPath := strconv.Itoa(idx)
	if mount.Type == "volume" && mount.Source != "" {
		subPath = mount.Source
	}

	return corev1.VolumeMount{
		Name:      "devpod",
		MountPath: mount.Target,
		SubPath:   fmt.Sprintf("devpod/%s", subPath),
	}
}

func getLabels(pod *corev1.Pod, rawLabels string) (map[string]string, error) {
	labels := map[string]string{}
	if pod.ObjectMeta.Labels != nil {
		for k, v := range pod.ObjectMeta.Labels {
			labels[k] = v
		}
	}
	if rawLabels != "" {
		extraLabels, err := parseLabels(rawLabels)
		if err != nil {
			return nil, fmt.Errorf("parse labels: %w", err)
		}
		for k, v := range extraLabels {
			labels[k] = v
		}
	}
	// make sure we don't overwrite the devpod labels
	for k, v := range DevPodLabels {
		labels[k] = v
	}

	return labels, nil
}

func getNodeSelector(pod *corev1.Pod, rawNodeSelector string) (map[string]string, error) {
	nodeSelector := map[string]string{}
	if pod.Spec.NodeSelector != nil {
		for k, v := range pod.Spec.NodeSelector {
			nodeSelector[k] = v
		}
	}

	if rawNodeSelector != "" {
		selector, err := parseLabels(rawNodeSelector)
		if err != nil {
			return nil, fmt.Errorf("parsing node selector: %w", err)
		}
		for k, v := range selector {
			nodeSelector[k] = v
		}
	}

	return nodeSelector, nil
}

func (k *kubernetesDriver) StartDevContainer(ctx context.Context, workspaceId string) error {
	workspaceId = getID(workspaceId)
	_, containerInfo, err := k.getDevContainerPvc(ctx, workspaceId)
	if err != nil {
		return err
	} else if containerInfo == nil {
		return fmt.Errorf("persistent volume '%s' not found", workspaceId)
	}

	return k.runContainer(
		ctx,
		workspaceId,
		containerInfo.Options,
		false,
	)
}

func getID(workspaceID string) string {
	return "devpod-" + workspaceID
}

