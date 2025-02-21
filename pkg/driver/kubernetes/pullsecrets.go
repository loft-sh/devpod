package kubernetes

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/pkg/dockercredentials"
	perrors "github.com/pkg/errors"
	k8sv1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *KubernetesDriver) EnsurePullSecret(
	ctx context.Context,
	pullSecretName string,
	dockerImage string,
) (bool, error) {
	k.Log.Debugf("Ensure pull secrets")

	host, err := GetRegistryFromImageName(dockerImage)
	if err != nil {
		return false, fmt.Errorf("get registry from image name: %w", err)
	}

	dockerCredentials, err := dockercredentials.GetAuthConfig(host)
	if err != nil || dockerCredentials == nil || dockerCredentials.Username == "" || dockerCredentials.Secret == "" {
		k.Log.Debugf("Couldn't retrieve credentials for registry: %s", host)
		return false, nil
	}

	if k.secretExists(ctx, pullSecretName) {
		if !k.shouldRecreateSecret(ctx, dockerCredentials, pullSecretName, host) {
			k.Log.Debugf("Pull secret '%s' already exists and is up to date", pullSecretName)
			return true, nil
		}

		k.Log.Debugf("Pull secret '%s' already exists, but is outdated. Recreating...", pullSecretName)
		err := k.DeletePullSecret(ctx, pullSecretName)
		if err != nil {
			return false, err
		}
	}

	err = k.createPullSecret(ctx, pullSecretName, dockerCredentials)
	if err != nil {
		return false, err
	}

	k.Log.Infof("Pull secret '%s' created", pullSecretName)
	return true, nil
}

func (k *KubernetesDriver) ReadSecretContents(
	ctx context.Context,
	pullSecretName string,
	host string,
) (string, error) {
	secret, err := k.client.Client().CoreV1().Secrets(k.namespace).Get(ctx, pullSecretName, metav1.GetOptions{})
	if err != nil {
		return "", perrors.Wrap(err, "get secret")
	}

	return DecodeAuthTokenFromPullSecret(secret, host)
}

func (k *KubernetesDriver) DeletePullSecret(
	ctx context.Context,
	pullSecretName string) error {
	if !k.secretExists(ctx, pullSecretName) {
		return nil
	}

	err := k.client.Client().CoreV1().Secrets(k.namespace).Delete(ctx, pullSecretName, metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return perrors.Wrap(err, "delete pull secret")
	}

	return nil
}

func (k *KubernetesDriver) shouldRecreateSecret(ctx context.Context, dockerCredentials *dockercredentials.Credentials, pullSecretName, host string) bool {
	existingAuthToken, err := k.ReadSecretContents(ctx, pullSecretName, host)
	if err != nil {
		return true
	}
	return existingAuthToken != dockerCredentials.AuthToken()
}

func (k *KubernetesDriver) secretExists(
	ctx context.Context,
	pullSecretName string,
) bool {
	_, err := k.client.Client().CoreV1().Secrets(k.namespace).Get(ctx, pullSecretName, metav1.GetOptions{})
	return err == nil
}

func (k *KubernetesDriver) createPullSecret(
	ctx context.Context,
	pullSecretName string,
	dockerCredentials *dockercredentials.Credentials,
) error {
	authToken := dockerCredentials.AuthToken()
	email := "noreply@loft.sh"

	encodedSecretData, err := PreparePullSecretData(dockerCredentials.ServerURL, authToken, email)
	if err != nil {
		return perrors.Wrap(err, "prepare pull secret data")
	}

	_, err = k.client.Client().CoreV1().Secrets(k.namespace).Create(ctx, &k8sv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: pullSecretName,
		},
		Type: k8sv1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": []byte(encodedSecretData),
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return perrors.Wrap(err, "create pull secret")
	}

	return nil
}
