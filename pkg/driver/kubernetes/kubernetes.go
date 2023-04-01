package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/pkg/compose"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"io"
	corev1 "k8s.io/api/core/v1"
	"os/exec"
	"strings"
)

func NewKubernetesDriver(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) driver.Driver {
	kubectl := "kubectl"
	if workspaceInfo.Agent.Kubernetes.Path != "" {
		kubectl = workspaceInfo.Agent.Kubernetes.Path
	}

	return &kubernetesDriver{
		kubectl: kubectl,

		kubeConfig: workspaceInfo.Agent.Kubernetes.Config,
		context:    workspaceInfo.Agent.Kubernetes.Context,
		namespace:  workspaceInfo.Agent.Kubernetes.Namespace,

		config: workspaceInfo.Agent.Kubernetes,
		Log:    log,
	}
}

type kubernetesDriver struct {
	kubectl string

	kubeConfig string
	namespace  string
	context    string

	config provider2.ProviderKubernetesDriverConfig
	Log    log.Logger
}

func (k *kubernetesDriver) FindDevContainer(ctx context.Context, labels []string) (*config.ContainerDetails, error) {
	id, err := k.getID(labels)
	if err != nil {
		return nil, errors.Wrap(err, "get name")
	}

	pvc, containerInfo, err := k.getDevContainerPvc(ctx, id)
	if err != nil {
		return nil, err
	}

	return k.infoFromObject(ctx, pvc, containerInfo)
}

func (k *kubernetesDriver) getDevContainerPvc(ctx context.Context, id string) (*corev1.PersistentVolumeClaim, *DevContainerInfo, error) {
	// try to find pvc
	out, err := k.buildCmd(ctx, []string{"get", "pvc", id, "--ignore-not-found", "-o", "json"}).Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return nil, nil, fmt.Errorf("find pvc: %s", strings.TrimSpace(string(exitError.Stderr)))
		}

		return nil, nil, errors.Wrap(err, "find pvc")
	} else if len(out) == 0 {
		return nil, nil, nil
	}

	// try to unmarshal pod
	pvc := &corev1.PersistentVolumeClaim{}
	err = json.Unmarshal(out, pvc)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unmarshal pvc")
	} else if pvc.Annotations == nil || pvc.Annotations[DevContainerInfoAnnotation] == "" {
		return nil, nil, fmt.Errorf("pvc is missing dev container info annotation")
	}

	// get container info
	containerInfo := &DevContainerInfo{}
	err = json.Unmarshal([]byte(pvc.GetAnnotations()[DevContainerInfoAnnotation]), containerInfo)
	if err != nil {
		return nil, nil, errors.Wrap(err, "decode dev container info")
	}

	return pvc, containerInfo, nil
}

func (k *kubernetesDriver) infoFromObject(ctx context.Context, pvc *corev1.PersistentVolumeClaim, containerInfo *DevContainerInfo) (*config.ContainerDetails, error) {
	if pvc == nil {
		return nil, nil
	}

	// merge env
	env := containerInfo.ImageDetails.Config.Env
	for k, v := range containerInfo.MergedConfig.ContainerEnv {
		env = append(env, k+"="+v)
	}

	// merge labels
	labels := map[string]string{}
	for k, v := range containerInfo.ImageDetails.Config.Labels {
		labels[k] = v
	}
	for k, v := range config.ListToObject(containerInfo.Labels) {
		labels[k] = v
	}

	// check pod
	pod, err := k.waitPodRunning(ctx, pvc.Name)
	if err != nil {
		return nil, err
	}

	// determine status
	status := "exited"
	if pod != nil {
		status = "running"
	}

	// check started
	startedAt := pvc.CreationTimestamp.String()
	if pod != nil {
		startedAt = pod.CreationTimestamp.String()
	}

	return &config.ContainerDetails{
		Id:      pvc.Name,
		Created: pvc.CreationTimestamp.String(),
		State: config.ContainerDetailsState{
			Status:    status,
			StartedAt: startedAt,
		},
		Config: config.ContainerDetailsConfig{
			Image:  containerInfo.ImageName,
			User:   containerInfo.ImageDetails.Config.User,
			Env:    env,
			Labels: labels,
		},
	}, nil
}

func (k *kubernetesDriver) StopDevContainer(ctx context.Context, id string) error {
	// delete pod
	out, err := k.buildCmd(ctx, []string{"delete", "po", id, "--ignore-not-found"}).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "delete pod: %s", string(out))
	}

	return nil
}

func (k *kubernetesDriver) DeleteDevContainer(ctx context.Context, id string) error {
	// delete pod
	err := k.deletePod(ctx, id)
	if err != nil {
		return err
	}

	// delete pvc
	out, err := k.buildCmd(ctx, []string{"delete", "pvc", id, "--ignore-not-found", "--grace-period=5"}).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "delete pvc: %s", string(out))
	}

	return nil
}

func (k *kubernetesDriver) deletePod(ctx context.Context, podName string) error {
	out, err := k.buildCmd(ctx, []string{"delete", "po", podName, "--ignore-not-found", "--grace-period=10"}).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "delete pod: %s", string(out))
	}

	return nil
}

func (k *kubernetesDriver) CommandDevContainer(ctx context.Context, id, user, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	args := []string{"exec", "-c", "devpod"}
	if stdin != nil {
		args = append(args, "-i")
	}
	args = append(args, id)
	if user != "" && user != "root" {
		args = append(args, "--", "su", user, "-c", command)
	} else {
		args = append(args, "--", "sh", "-c", command)
	}

	return k.runCommand(ctx, args, stdin, stdout, stderr)
}

func (k *kubernetesDriver) buildCmd(ctx context.Context, args []string) *exec.Cmd {
	newArgs := []string{}
	if k.namespace != "" {
		newArgs = []string{"--namespace", k.namespace}
	}
	if k.kubeConfig != "" {
		newArgs = append(args, "--kubeconfig", k.kubeConfig)
	}
	if k.context != "" {
		newArgs = append(args, "--context", k.context)
	}
	newArgs = append(newArgs, args...)
	k.Log.Debugf("Run command: %s %s", k.kubectl, strings.Join(args, " "))
	return exec.CommandContext(ctx, k.kubectl, args...)
}

func (k *kubernetesDriver) runCommand(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	return k.runCommandWithDir(ctx, "", args, stdin, stdout, stderr)
}

func (k *kubernetesDriver) runCommandWithDir(ctx context.Context, dir string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmd := k.buildCmd(ctx, args)
	cmd.Dir = dir
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func (k *kubernetesDriver) InspectImage(ctx context.Context, imageName string) (*config.ImageDetails, error) {
	imageConfig, _, err := image.GetImageConfig(imageName)
	if err != nil {
		return nil, err
	}

	return &config.ImageDetails{
		Id: imageName,
		Config: config.ImageDetailsConfig{
			User:       imageConfig.Config.User,
			Env:        imageConfig.Config.Env,
			Labels:     imageConfig.Config.Labels,
			Entrypoint: imageConfig.Config.Entrypoint,
			Cmd:        imageConfig.Config.Cmd,
		},
	}, nil
}

func (k *kubernetesDriver) ComposeHelper() (*compose.ComposeHelper, error) {
	return nil, fmt.Errorf("docker compose is currently not supported with Kubernetes")
}
