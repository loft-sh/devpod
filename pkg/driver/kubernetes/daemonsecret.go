package kubernetes

import (
	"context"
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DaemonConfigKey = "daemon_config"

func (k *KubernetesDriver) EnsureDaemonConfigSecret(
	ctx context.Context,
	secretName string,
	data string,
) error {
	k.Log.Debugf("Ensure daemon config secret")

	if k.secretExists(ctx, secretName) {
		if !k.shouldRecreateDaemonConfigSecret(ctx, data, secretName) {
			k.Log.Debugf("Daemon config secret '%s' already exists and is up to date", secretName)
			return nil
		}

		k.Log.Debugf("Daemon config secret '%s' already exists, but is outdated. Recreating...", secretName)
		err := k.DeleteSecret(ctx, secretName)
		if err != nil {
			return err
		}
	}

	_, err := k.client.Client().CoreV1().Secrets(k.namespace).Create(ctx, &k8sv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Type: k8sv1.SecretTypeOpaque,
		Data: map[string][]byte{DaemonConfigKey: []byte(data)},
	}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create daemon config secret: %w", err)
	}

	k.Log.Infof("Daemon config secret '%s' created", secretName)
	return nil
}
func (k *KubernetesDriver) shouldRecreateDaemonConfigSecret(ctx context.Context, newData string, secretName string) bool {
	secret, err := k.client.Client().CoreV1().Secrets(k.namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return true
	}

	daemonConfigBytes, ok := secret.Data[DaemonConfigKey]
	if !ok {
		return true
	}

	return string(daemonConfigBytes) != newData
}

func (k *KubernetesDriver) DeleteDaemonConfigSecret(
	ctx context.Context,
	secretName string) error {
	if !k.secretExists(ctx, secretName) {
		return nil
	}

	err := k.client.Client().CoreV1().Secrets(k.namespace).Delete(ctx, secretName, metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("delete daemon config secret: %w", err)
	}

	return nil
}
