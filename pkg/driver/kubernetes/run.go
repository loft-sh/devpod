package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	config2 "github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver/docker"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"strings"
)

const DevContainerInfoAnnotation = "devpod.sh/info"

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

	// create persistent volume claim
	err = k.createPersistentVolumeClaim(ctx, id, parsedConfig, mergedConfig, imageName, workspaceMount, labels, imageDetails)
	if err != nil {
		return err
	}

	// create dev container
	err = k.runContainer(ctx, id, parsedConfig, mergedConfig, imageName, workspaceMount, labels, imageDetails, true)
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
	labels []string,
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

	// create the pod manifest
	entrypoint, args := docker.GetContainerEntrypointAndArgs(mergedConfig, imageDetails)
	podRaw, err := json.Marshal(&corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
		},
		Spec: corev1.PodSpec{
			InitContainers: initContainer,
			Containers: []corev1.Container{
				{
					Name:         "devpod",
					Image:        imageName,
					Command:      []string{entrypoint},
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
			RestartPolicy:                corev1.RestartPolicyNever,
			AutomountServiceAccountToken: &[]bool{false}[0],
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
	})
	if err != nil {
		return err
	}

	// create the pod
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
			k.Log.Infof("Copy %s into DevContainer %s", copyMount.Source, copyMount.Target)
			buf := &bytes.Buffer{}
			err = k.runCommandWithDir(ctx, filepath.Dir(parsedConfig.Origin), []string{"cp", "-c", "devpod", strings.TrimRight(copyMount.Source, "/") + "/.", fmt.Sprintf("%s:%s", id, strings.TrimRight(copyMount.Target, "/"))}, nil, buf, buf)
			if err != nil {
				return errors.Wrap(err, "copy to devcontainer")
			}
		}
	}

	return nil
}

func getVolumeMount(idx int, mount *config.Mount) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      "devpod",
		MountPath: mount.Target,
		SubPath:   fmt.Sprintf("devpod/%d", idx),
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
		containerInfo.Labels,
		containerInfo.ImageDetails,
		false,
	)
}
