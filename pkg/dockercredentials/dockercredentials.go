package dockercredentials

import (
	"encoding/json"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/loft-sh/devpod/pkg/docker"
	"os"
)

// Credentials holds the information shared between docker and the credentials store.
type Credentials struct {
	ServerURL string
	Username  string
	Secret    string
}

func ConfigureCredentials(dockerCredentials string) (string, error) {
	dockerConfig, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}
	defer dockerConfig.Close()

	_, err = dockerConfig.WriteString(dockerCredentials)
	if err != nil {
		return "", err
	}

	err = os.Setenv("DOCKER_CONFIG", dockerConfig.Name())
	if err != nil {
		return "", err
	}

	return dockerConfig.Name(), nil
}

func GetAuthConfigs() (map[string]types.AuthConfig, error) {
	dockerConfig, err := docker.LoadDockerConfig()
	if err != nil {
		return nil, err
	}

	return dockerConfig.GetAuthConfigs(), nil
}

func GetFilledCredentials() ([]byte, error) {
	dockerConfig, err := docker.LoadDockerConfig()
	if err != nil {
		return nil, err
	}

	authConfigs := dockerConfig.GetAuthConfigs()
	for key, config := range authConfigs {
		host := config.ServerAddress
		if host == "registry-1.docker.io" {
			host = "https://index.docker.io/v1/"
		}
		ac, err := dockerConfig.GetAuthConfig(host)
		if err != nil {
			return nil, err
		}

		config.Username = ac.Username
		config.Password = ac.Password
		config.IdentityToken = ac.IdentityToken
		authConfigs[key] = config
	}

	dockerFile := &configfile.ConfigFile{
		AuthConfigs: authConfigs,
	}
	dockerFileRaw, err := json.Marshal(dockerFile)
	if err != nil {
		return nil, err
	}

	return dockerFileRaw, nil
}

func GetAuthConfig(host string) (*Credentials, error) {
	dockerConfig, err := docker.LoadDockerConfig()
	if err != nil {
		return nil, err
	}

	if host == "registry-1.docker.io" {
		host = "https://index.docker.io/v1/"
	}
	ac, err := dockerConfig.GetAuthConfig(host)
	if err != nil {
		return nil, err
	}

	if ac.IdentityToken != "" {
		return &Credentials{
			ServerURL: host,
			Secret:    ac.IdentityToken,
		}, nil
	}

	return &Credentials{
		ServerURL: host,
		Username:  ac.Username,
		Secret:    ac.Password,
	}, nil
}
