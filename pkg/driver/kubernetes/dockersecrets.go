package kubernetes

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	perrors "github.com/pkg/errors"
	k8sv1 "k8s.io/api/core/v1"
)

// DockerConfigJSON represents a local docker auth config file
// for pulling images.
type DockerConfigJSON struct {
	Auths DockerConfig `json:"auths"`
}

// DockerConfig represents the config file used by the docker CLI.
// This config that represents the credentials that should be used
// when pulling images from specific image repositories.
type DockerConfig map[string]DockerConfigEntry

// DockerConfigEntry holds the user information that grant the access to docker registry
type DockerConfigEntry struct {
	Auth  string `json:"auth"`
	Email string `json:"email"`
}

func PreparePullSecretData(registryURL, authToken, email string) (string, error) {
	dockerConfig := &DockerConfigJSON{
		Auths: DockerConfig{
			registryURL: newDockerConfigEntry(authToken, email),
		},
	}

	pullSecretData, err := toPullSecretData(dockerConfig)
	if err != nil {
		return "", perrors.Wrap(err, "new pull secret")
	}

	return pullSecretData, nil
}

func newDockerConfigEntry(authToken, email string) DockerConfigEntry {
	return DockerConfigEntry{
		Auth:  base64.StdEncoding.EncodeToString([]byte(authToken)),
		Email: email,
	}
}

func toPullSecretData(dockerConfig *DockerConfigJSON) (string, error) {
	data, err := json.Marshal(dockerConfig)
	if err != nil {
		return "", perrors.Wrap(err, "marshal docker config")
	}

	return string(data), nil
}

func DecodeAuthTokenFromPullSecret(secret *k8sv1.Secret, host string) (string, error) {
	dockerConfigBytes, ok := secret.Data[k8sv1.DockerConfigJsonKey]
	if !ok {
		return "", fmt.Errorf("could not find %s in secret data", k8sv1.DockerConfigJsonKey)
	}

	var dockerConfig DockerConfigJSON
	err := json.Unmarshal(dockerConfigBytes, &dockerConfig)
	if err != nil {
		return "", perrors.Wrap(err, "unmarshal docker config")
	}

	auth, ok := dockerConfig.Auths[host]
	if !ok {
		return "", fmt.Errorf("no auth found for host: %s", host)
	}

	decodedAuthToken, err := base64.StdEncoding.DecodeString(auth.Auth)
	if err != nil {
		return "", perrors.Wrap(err, "decode auth token")
	}

	return string(decodedAuthToken), nil
}
