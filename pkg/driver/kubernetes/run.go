package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const DevContainerName = "devpod"
const InitContainerName = "devpod-init"

const (
	DevPodCreatedLabel      = "devpod.sh/created"
	DevPodWorkspaceLabel    = "devpod.sh/workspace"
	DevPodWorkspaceUIDLabel = "devpod.sh/workspace-uid"

	DevPodInfoAnnotation                   = "devpod.sh/info"
	DevPodLastAppliedAnnotation            = "devpod.sh/last-applied-configuration"
	ClusterAutoscalerSaveToEvictAnnotation = "cluster-autoscaler.kubernetes.io/safe-to-evict"
)

var ExtraDevPodLabels = map[string]string{
	DevPodCreatedLabel: "true",
}

type DevContainerInfo struct {
	WorkspaceID string
	Options     *driver.RunOptions
}

func (k *KubernetesDriver) RunDevContainer(
	ctx context.Context,
	workspaceId string,
	options *driver.RunOptions,
) error {
	k.Log.Debugf("Running devcontainer for workspace '%s'", workspaceId)
	workspaceId = getID(workspaceId)

	// namespace
	if k.namespace != "" && k.options.CreateNamespace == "true" {
		err := k.createNamespace(ctx)
		if err != nil {
			return err
		}
	}

	// check if persistent volume claim already exists
	initialize := false
	pvc, containerInfo, err := k.getDevContainerPvc(ctx, workspaceId)
	if err != nil {
		return err
	} else if pvc == nil {
		if options == nil {
			return fmt.Errorf("no options provided and no persistent volume claim found for workspace '%s'", workspaceId)
		}

		// create persistent volume claim
		err = k.createPersistentVolumeClaim(ctx, workspaceId, options)
		if err != nil {
			return err
		}

		initialize = true
	}

	// reuse driver.RunOptions from existing workspace if none provided
	if options == nil && containerInfo != nil && containerInfo.Options != nil {
		options = containerInfo.Options
	}

	// create dev container
	err = k.runContainer(ctx, workspaceId, options, initialize)
	if err != nil {
		return err
	}

	return nil
}

func (k *KubernetesDriver) runContainer(
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
	if k.options.WorkspaceVolumeMount != "" {
		// Ensure workspace volume mount option is parent or same dir as workspace mount
		rel, err := filepath.Rel(k.options.WorkspaceVolumeMount, mount.Target)
		if err != nil {
			k.Log.Warn("Relative filepath: %v", err)
		} else if strings.HasPrefix(rel, "..") {
			k.Log.Warnf("Workspace volume mount needs to be the same as the workspace mount or a parent, skipping option. WorkspaceVolumeMount: %s, MountTarget: %s", k.options.WorkspaceVolumeMount, mount.Target)
		} else {
			mount.Target = k.options.WorkspaceVolumeMount
			k.Log.Debugf("Using workspace volume mount: %s", k.options.WorkspaceVolumeMount)
		}
	}

	// read pod template
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
	if len(k.options.PodManifestTemplate) > 0 {
		k.Log.Debugf("trying to get pod template manifest from %s", k.options.PodManifestTemplate)
		pod, err = getPodTemplate(k.options.PodManifestTemplate)
		if err != nil {
			return err
		}
	}

	// get init containers
	initContainers, err := k.getInitContainers(options, pod, initialize)
	if err != nil {
		return errors.Wrap(err, "build init container")
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
	daemonConfig := ""
	for k, v := range options.Env {
		// filter out daemon config, that's going to be mounted through a secret
		if k == config.WorkspaceDaemonConfigExtraEnvVar {
			daemonConfig = v
			continue
		}
		envVars = append(envVars, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	// service account
	serviceAccount := ""
	if k.options.ServiceAccount != "" {
		serviceAccount = k.options.ServiceAccount

		// create service account
		err = k.createServiceAccount(ctx, id, serviceAccount)
		if err != nil {
			return fmt.Errorf("create service account: %w", err)
		}
	}

	// labels
	labels, err := getLabels(pod, k.options.Labels)
	if err != nil {
		return err
	}
	labels[DevPodWorkspaceUIDLabel] = options.UID

	// node selector
	nodeSelector, err := getNodeSelector(pod, k.options.NodeSelector)
	if err != nil {
		return err
	}

	// parse resources
	resources := corev1.ResourceRequirements{}
	if len(pod.Spec.Containers) > 0 {
		resources = pod.Spec.Containers[0].Resources
	}
	if k.options.Resources != "" {
		resources = parseResources(k.options.Resources, k.Log)
	}

	// ensure daemon config secret
	daemonConfigSecretName := ""
	if daemonConfig != "" {
		daemonConfigSecretName = getDaemonSecretName(id)
		err = k.EnsureDaemonConfigSecret(ctx, daemonConfigSecretName, daemonConfig)
		if err != nil {
			return err
		}
	}

	// ensure pull secrets
	pullSecretsCreated := false
	if k.options.KubernetesPullSecretsEnabled == "true" {
		pullSecretsCreated, err = k.EnsurePullSecret(ctx, getPullSecretsName(id), options.Image)
		if err != nil {
			return err
		}
	}

	// create the pod manifest
	pod.ObjectMeta.Name = id
	pod.ObjectMeta.Labels = labels

	pod.Spec.ServiceAccountName = serviceAccount
	pod.Spec.NodeSelector = nodeSelector
	pod.Spec.InitContainers = initContainers
	pod.Spec.Containers = getContainers(pod, options.Image, options.Entrypoint, options.Cmd, envVars, volumeMounts, capabilities, resources, options.Privileged, k.options.StrictSecurity, daemonConfigSecretName)
	pod.Spec.Volumes = getVolumes(pod, id, daemonConfigSecretName)
	// avoids a problem where attaching volumes with large repositories would cause an extremely long pod startup time
	// because changing the ownership of all files takes longer than the kubelet expects it to
	if pod.Spec.SecurityContext == nil {
		pod.Spec.SecurityContext = &corev1.PodSecurityContext{
			FSGroupChangePolicy: ptr.To(corev1.FSGroupChangeOnRootMismatch),
		}
	}
	if k.options.KubernetesPullSecretsEnabled == "true" && pullSecretsCreated {
		pod.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: getPullSecretsName(id)}}
	}
	pod.Spec.RestartPolicy = corev1.RestartPolicyNever
	// try to get existing pod
	existingPod, err := k.getPod(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "get pod: %s", id)
	}

	if existingPod != nil {
		existingOptions := &provider2.ProviderKubernetesDriverConfig{}
		err := json.Unmarshal([]byte(existingPod.GetAnnotations()[DevPodLastAppliedAnnotation]), existingOptions)
		if err != nil {
			k.Log.Errorf("Error unmarshalling existing provider options, continuing...: %s", err)
		}

		// Nothing changed, can safely return
		if optionsEqual(existingOptions, k.options) {
			k.Log.Infof("Pod '%s' already exists and nothing changed, skipping update", existingPod.Name)
			return nil
		}

		// Stop the current pod
		k.Log.Debug("Provider options changed")
		err = k.waitPodDeleted(ctx, id)
		if err != nil {
			return errors.Wrapf(err, "stop devcontainer: %s", id)
		}
	}

	err = k.runPod(ctx, id, pod)
	if err != nil {
		return err
	}

	return nil
}

func (k *KubernetesDriver) runPod(ctx context.Context, id string, pod *corev1.Pod) error {
	var err error

	// set configuration before creating the pod
	lastAppliedConfigRaw, err := json.Marshal(k.options)
	if err != nil {
		return errors.Wrap(err, "marshal last applied config")
	}

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	pod.Annotations[DevPodLastAppliedAnnotation] = string(lastAppliedConfigRaw)
	pod.Annotations[ClusterAutoscalerSaveToEvictAnnotation] = "false"

	// marshal the pod
	podRaw, err := json.Marshal(pod)
	if err != nil {
		return err
	}

	k.Log.Debugf("Create pod with: %s", string(podRaw))

	// create the pod
	k.Log.Infof("Create Pod '%s'", id)
	_, err = k.client.Client().CoreV1().Pods(k.namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create pod: %w", err)
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
	strictSecurity string,
	daemonConfigSecretName string,
) []corev1.Container {
	if daemonConfigSecretName != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "devpod-daemon-config",
			MountPath: "/var/run/secrets/devpod",
		})
	}
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

	if strictSecurity == "true" {
		devPodContainer.SecurityContext = nil
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
		devPodContainer.ImagePullPolicy = existingDevPodContainer.ImagePullPolicy

		if devPodContainer.SecurityContext == nil && existingDevPodContainer.SecurityContext != nil {
			devPodContainer.SecurityContext = existingDevPodContainer.SecurityContext
		}
	}
	retContainers = append(retContainers, devPodContainer)

	return retContainers
}

func getVolumes(pod *corev1.Pod, id string, daemonConfigSecretName string) []corev1.Volume {
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

	if daemonConfigSecretName != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "devpod-daemon-config",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: daemonConfigSecretName,
				},
			},
		})
	}

	if pod.Spec.Volumes != nil {
		volumes = append(volumes, pod.Spec.Volumes...)
	}

	return volumes
}

func getVolumeMount(idx int, mount *config.Mount) corev1.VolumeMount {
	subPath := strconv.Itoa(idx)
	if mount.Type == "volume" && mount.Source != "" {
		subPath = strings.TrimPrefix(mount.Source, "/")
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
	for k, v := range ExtraDevPodLabels {
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

func (k *KubernetesDriver) StartDevContainer(ctx context.Context, workspaceId string) error {
	k.Log.Debugf("Starting devcontainer for workspace '%s'", workspaceId)
	defer k.Log.Debugf("Done starting devcontainer for workspace '%s'", workspaceId)

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

func getPullSecretsName(workspaceID string) string {
	return fmt.Sprintf("devpod-pull-secret-%s", workspaceID)
}

func getDaemonSecretName(workspaceID string) string {
	return fmt.Sprintf("devpod-daemon-secret-%s", workspaceID)
}

func optionsEqual(a, b *provider2.ProviderKubernetesDriverConfig) bool {
	// copy a and b and the compare them without the context, config, namespace and podTimeout
	aCopy := *a
	aCopy.KubernetesContext = ""
	aCopy.KubernetesConfig = ""
	aCopy.KubernetesNamespace = ""
	aCopy.PodTimeout = ""

	bCopy := *b
	bCopy.KubernetesContext = ""
	bCopy.KubernetesConfig = ""
	bCopy.KubernetesNamespace = ""
	bCopy.PodTimeout = ""
	return aCopy == bCopy
}

func (k *KubernetesDriver) createNamespace(ctx context.Context) error {
	_, err := k.client.Client().CoreV1().Namespaces().Get(ctx, k.namespace, metav1.GetOptions{})
	if kerrors.IsNotFound(err) || kerrors.IsForbidden(err) {
		k.Log.Infof("Create namespace '%s'", k.namespace)
		_, err := k.client.Client().CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: k.namespace,
			},
		}, metav1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) && !kerrors.IsForbidden(err) {
			return fmt.Errorf("create namespace: %w", err)
		}
	}

	return nil
}
