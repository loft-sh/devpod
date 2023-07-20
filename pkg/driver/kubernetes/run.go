package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	config2 "github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver/docker"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DevContainerInfoAnnotation = "devpod.sh/info"

var DevPodLabels = map[string]string{
	"devpod.sh/created": "true",
}

type DevContainerInfo struct {
	ParsedConfig   *config.DevContainerConfig
	MergedConfig   *config.MergedDevContainerConfig
	ImageDetails   *config.ImageDetails
	ImageName      string
	WorkspaceMount string
	Labels         []string
}

func (k *kubernetesDriver) RunDevContainer(
	ctx context.Context,
	parsedConfig *config.DevContainerConfig,
	mergedConfig *config.MergedDevContainerConfig,
	imageName,
	workspaceMount string,
	labels []string,
	ide string,
	ideOptions map[string]config2.OptionValue,
	imageDetails *config.ImageDetails,
) error {
	id, err := k.getID(labels)
	if err != nil {
		return err
	}

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
	pvc, _, err := k.getDevContainerPvc(ctx, id)
	if err != nil {
		return err
	} else if pvc == nil {
		// create persistent volume claim
		err = k.createPersistentVolumeClaim(ctx, id, parsedConfig, mergedConfig, imageName, workspaceMount, labels, imageDetails)
		if err != nil {
			return err
		}

		initialize = true
	}

	// create dev container
	err = k.runContainer(ctx, id, parsedConfig, mergedConfig, imageName, workspaceMount, imageDetails, initialize)
	if err != nil {
		return err
	}

	return nil
}

func (k *kubernetesDriver) runContainer(
	ctx context.Context,
	id string,
	parsedConfig *config.DevContainerConfig,
	mergedConfig *config.MergedDevContainerConfig,
	imageName string,
	workspaceMount string,
	imageDetails *config.ImageDetails,
	initialize bool,
) (err error) {
	// get workspace mount
	mount := config.ParseMount(workspaceMount)
	if mount.Target == "" {
		return fmt.Errorf("workspace mount target is empty")
	}

	// get init container
	var initContainer []corev1.Container
	if initialize {
		initContainer, err = k.getInitContainer(mergedConfig, imageName)
		if err != nil {
			return errors.Wrap(err, "build init container")
		}
	}

	// loop over volume mounts
	copyFromLocal := []*config.Mount{&mount}
	volumeMounts := []corev1.VolumeMount{getVolumeMount(0, &mount)}
	for idx, mount := range mergedConfig.Mounts {
		volumeMount := getVolumeMount(idx+1, mount)
		if mount.Type == "bind" {
			copyFromLocal = append(copyFromLocal, mount)
			volumeMounts = append(volumeMounts, volumeMount)
		} else if mount.Type == "volume" {
			volumeMounts = append(volumeMounts, volumeMount)
		} else {
			k.Log.Warnf("Unsupported mount type '%s' in mount '%s', will skip", mount.Type, mount.String())
		}
	}

	// capabilities
	var capabilities *corev1.Capabilities
	if len(mergedConfig.CapAdd) > 0 {
		capabilities = &corev1.Capabilities{}
		for _, cap := range mergedConfig.CapAdd {
			capabilities.Add = append(capabilities.Add, corev1.Capability(cap))
		}
	}

	// env vars
	envVars := []corev1.EnvVar{}
	for k, v := range mergedConfig.ContainerEnv {
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

	labels := map[string]string{}
	if k.config.Labels != "" {
		extraLabels, err := parseLabels(k.config.Labels)
		if err != nil {
			return fmt.Errorf("parse labels: %w", err)
		}
		for k, v := range extraLabels {
			labels[k] = v
		}
	}
	// make sure we don't overwrite the devpod labels
	for k, v := range DevPodLabels {
		labels[k] = v
	}

	// create the pod manifest
	entrypoint, args := docker.GetContainerEntrypointAndArgs(mergedConfig, imageDetails)

	var podTemplate *corev1.Pod
	devPodCustomizations := config.GetDevPodCustomizations(parsedConfig)
	if len(devPodCustomizations.PodManifestTemplate) > 0 {
		podManifestTemplatePath := filepath.Join(mount.Source, devPodCustomizations.PodManifestTemplate[0])
		podTemplate, err = getPodTemplate(podManifestTemplatePath)
		if err != nil {
			return err
		}
	}

	var pod *corev1.Pod
	if podTemplate != nil {
		pod = podTemplate
		pod.ObjectMeta.Name = id
		pod.ObjectMeta.Namespace = ""
		pod.ObjectMeta.Labels = map[string]string{}
		if len(podTemplate.ObjectMeta.Labels) > 0 {
			for k, v := range podTemplate.ObjectMeta.Labels {
				if _, ok := DevPodLabels[k]; !ok {
					// make sure we don't overwrite the devpod labels
					labels[k] = v
				}
			}
		}
		pod.Spec.ServiceAccountName = serviceAccount

		pod.Spec.InitContainers = append(initContainer, podTemplate.Spec.InitContainers...)
		pod.Spec.Containers[0].Name = "devpod"
		pod.Spec.Containers[0].Image = imageName
		pod.Spec.Containers[0].Command = []string{entrypoint}
		pod.Spec.Containers[0].Args = args
		pod.Spec.Containers[0].Env = append(envVars, podTemplate.Spec.Containers[0].Env...)
		pod.Spec.Containers[0].VolumeMounts = append(volumeMounts, podTemplate.Spec.Containers[0].VolumeMounts...)
		pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{
			Capabilities: capabilities,
			Privileged:   mergedConfig.Privileged,
			RunAsUser:    &[]int64{0}[0],
			RunAsGroup:   &[]int64{0}[0],
			RunAsNonRoot: &[]bool{false}[0],
		}
		pod.Spec.RestartPolicy = corev1.RestartPolicyNever
		pod.Spec.Volumes = append(
			[]corev1.Volume{
				{
					Name: "devpod",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: id,
						},
					},
				},
			},
			podTemplate.Spec.Volumes...,
		)
	} else {
		pod = &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   id,
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: serviceAccount,
				InitContainers:     initContainer,
				Containers: []corev1.Container{
					{
						Name:         "devpod",
						Image:        imageName,
						Command:      []string{entrypoint},
						Resources:    parseResources(k.config.Resources, k.Log),
						Args:         args,
						Env:          envVars,
						VolumeMounts: volumeMounts,
						SecurityContext: &corev1.SecurityContext{
							Capabilities: capabilities,
							Privileged:   mergedConfig.Privileged,
							RunAsUser:    &[]int64{0}[0],
							RunAsGroup:   &[]int64{0}[0],
							RunAsNonRoot: &[]bool{false}[0],
						},
					},
				},
				RestartPolicy: corev1.RestartPolicyNever,
				Volumes: []corev1.Volume{
					{
						Name: "devpod",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: id,
							},
						},
					},
				},
			},
		}
	}

	// parse node selector
	if k.config.NodeSelector != "" {
		pod.Spec.NodeSelector, err = parseLabels(k.config.NodeSelector)
		if err != nil {
			return fmt.Errorf("parsing node selector: %w", err)
		}
	}

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

	// copy local to pod
	if initialize {
		for _, copyMount := range copyFromLocal {
			// run kubectl
			sourcePath, err := transformPath(copyMount.Source)
			if err != nil {
				return fmt.Errorf("transform path %s: %w", copyMount.Source, err)
			}

			k.Log.Infof("Copy %s into DevContainer %s", copyMount.Source, copyMount.Target)
			buf := &bytes.Buffer{}
			err = k.runCommandWithDir(ctx, filepath.Dir(parsedConfig.Origin), []string{"cp", "-c", "devpod", sourcePath, fmt.Sprintf("%s:%s", id, strings.TrimRight(copyMount.Target, "/"))}, nil, buf, buf)
			if err != nil {
				return errors.Wrap(err, "copy to devcontainer")
			}
		}
	}

	return nil
}

// transform creates a windows compatible path https://github.com/kubernetes/kubectl/issues/1225
func transformPath(source string) (string, error) {
	if runtime.GOOS != "windows" {
		return strings.TrimRight(source, "/") + "/.", nil
	}

	var err error
	source = filepath.FromSlash(source)
	if !filepath.IsAbs(source) {
		source, err = filepath.Abs(source)
		if err != nil {
			return "", err
		}
	}

	volumeName := filepath.VolumeName(source)
	if volumeName == "" {
		return "", fmt.Errorf("no volume name in string: %s", source)
	}

	source = strings.TrimPrefix(source, volumeName)
	return source, nil
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

func (k *kubernetesDriver) StartDevContainer(ctx context.Context, id string, labels []string) error {
	_, containerInfo, err := k.getDevContainerPvc(ctx, id)
	if err != nil {
		return err
	} else if containerInfo == nil {
		return fmt.Errorf("persistent volume '%s' not found", id)
	}

	return k.runContainer(
		ctx,
		id,
		containerInfo.ParsedConfig,
		containerInfo.MergedConfig,
		containerInfo.ImageName,
		containerInfo.WorkspaceMount,
		containerInfo.ImageDetails,
		false,
	)
}
