package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/loft-sh/devpod/pkg/driver"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	perrors "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewKubernetesDriver constructs a struct capable of provisioning a workspace and it's resources using kubernetes
func NewKubernetesDriver(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) (driver.ReprovisioningDriver, error) {
	options := workspaceInfo.Agent.Kubernetes
	if options.KubernetesConfig != "" {
		log.Debugf("Use Kubernetes Config '%s'", options.KubernetesConfig)
	}
	if options.KubernetesContext != "" {
		log.Debugf("Use Kubernetes Context '%s'", options.KubernetesContext)
	}

	client, namespace, err := NewClient(options.KubernetesConfig, options.KubernetesContext)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}
	// Namespace can be defined in many ways, we first check the kube config, then the provider options KUBERNETES_NAMESPACE, then failing that the default "devpod"
	if namespace == "" || namespace == "default" || options.KubernetesNamespace != "" {
		log.Debugf("Using Explicit Kubernetes Namespace")
		namespace = options.KubernetesNamespace
	}
	log.Debugf("Use Kubernetes Namespace '%s'", namespace)

	return &KubernetesDriver{
		client:    client,
		namespace: namespace,

		options: &options,
		Log:     log,
	}, nil
}

type KubernetesDriver struct {
	namespace string

	client *Client

	options *provider2.ProviderKubernetesDriverConfig
	Log     log.Logger
}

func (k *KubernetesDriver) CanReprovision() bool {
	return true
}

func (k *KubernetesDriver) getDevContainerPvc(ctx context.Context, id string) (*corev1.PersistentVolumeClaim, *DevContainerInfo, error) {
	// try to find pvc
	pvc, err := k.client.Client().CoreV1().PersistentVolumeClaims(k.namespace).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil, nil
		}

		return nil, nil, err
	} else if pvc.Annotations == nil || pvc.Annotations[DevPodInfoAnnotation] == "" {
		return nil, nil, fmt.Errorf("pvc is missing dev container info annotation")
	}

	// get container info
	containerInfo := &DevContainerInfo{}
	err = json.Unmarshal([]byte(pvc.GetAnnotations()[DevPodInfoAnnotation]), containerInfo)
	if err != nil {
		return nil, nil, perrors.Wrap(err, "decode dev container info")
	}

	return pvc, containerInfo, nil
}

func (k *KubernetesDriver) StopDevContainer(ctx context.Context, workspaceId string) error {
	k.Log.Debugf("Stopping devcontainer for workspace '%s'", workspaceId)
	defer k.Log.Debugf("Done stopping devcontainer for workspace '%s'", workspaceId)

	workspaceId = getID(workspaceId)

	// delete pod
	err := k.waitPodDeleted(ctx, workspaceId)
	if err != nil {
		return perrors.Wrap(err, "delete pod")
	}

	return nil
}

func (k *KubernetesDriver) DeleteDevContainer(ctx context.Context, workspaceId string) error {
	k.Log.Debugf("Deleting devcontainer for workspace '%s'", workspaceId)
	defer k.Log.Debugf("Done deleting devcontainer for workspace '%s'", workspaceId)

	workspaceId = getID(workspaceId)

	// delete pod
	k.Log.Infof("Delete pod '%s'...", workspaceId)
	err := k.waitPodDeleted(ctx, workspaceId)
	if err != nil {
		return err
	}

	// delete pvc
	k.Log.Infof("Delete persistent volume claim '%s'...", workspaceId)
	err = k.client.Client().CoreV1().PersistentVolumeClaims(k.namespace).Delete(ctx, workspaceId, metav1.DeleteOptions{
		GracePeriodSeconds: &[]int64{5}[0],
	})
	if err != nil && !kerrors.IsNotFound(err) {
		return perrors.Wrap(err, "delete pvc")
	}

	// delete role binding & service account
	if k.options.ClusterRole != "" {
		k.Log.Infof("Delete role binding '%s'...", workspaceId)
		err = k.client.Client().RbacV1().RoleBindings(k.namespace).Delete(ctx, workspaceId, metav1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return perrors.Wrap(err, "delete role binding")
		}
	}

	// delete daemon config secret
	if k.secretExists(ctx, getDaemonSecretName(workspaceId)) {
		k.Log.Infof("Delete daemon config secret '%s'...", workspaceId)
		err := k.DeleteSecret(ctx, getDaemonSecretName(workspaceId))
		if err != nil {
			return err
		}
	}

	// delete pull secret
	if k.options.KubernetesPullSecretsEnabled != "" {
		k.Log.Infof("Delete pull secret '%s'...", workspaceId)
		err := k.DeleteSecret(ctx, getPullSecretsName(workspaceId))
		if err != nil {
			return err
		}
	}

	return nil
}

func (k *KubernetesDriver) CommandDevContainer(ctx context.Context, workspaceId, user, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	workspaceId = getID(workspaceId)

	var args []string
	if user != "" && user != "root" {
		args = []string{"su", user, "-c", command}
	} else {
		args = []string{"sh", "-c", command}
	}

	return k.client.Exec(ctx, &ExecStreamOptions{
		Pod:       workspaceId,
		Namespace: k.namespace,
		Container: "devpod",
		Command:   args,
		Stdin:     stdin,
		Stdout:    stdout,
		Stderr:    stderr,
	})
}

func (k *KubernetesDriver) GetDevContainerLogs(ctx context.Context, workspaceID string, stdout io.Writer, stderr io.Writer) error {
	workspaceID = getID(workspaceID)

	logs, err := k.client.Logs(ctx, k.namespace, workspaceID, "devpod", true)
	if err != nil {
		return perrors.Wrap(err, "get logs")
	}

	_, err = io.Copy(stdout, logs)
	if err != nil {
		return perrors.Wrap(err, "copy logs")
	}

	return nil
}
