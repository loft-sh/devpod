package dockercredentials

import (
	"bytes"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/loft-sh/devpod/pkg/docker"
	"os"
	"path/filepath"
)

// Credentials holds the information shared between docker and the credentials store.
type Credentials struct {
	ServerURL string
	Username  string
	Secret    string
}

func ConfigureCredentials(dockerCredentials string) (string, error) {
	dockerConfigDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", err
	}

	err = os.WriteFile(filepath.Join(dockerConfigDir, "config.json"), []byte(dockerCredentials), os.ModePerm)
	if err != nil {
		return "", err
	}

	err = os.Setenv("DOCKER_CONFIG", dockerConfigDir)
	if err != nil {
		return "", err
	}

	return dockerConfigDir, nil
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

		config.ServerAddress = host
		config.Username = ac.Username
		config.Password = ac.Password
		config.IdentityToken = ac.IdentityToken
		authConfigs[key] = config
	}

	dockerFile := &configfile.ConfigFile{
		AuthConfigs: authConfigs,
	}

	buf := &bytes.Buffer{}
	err = dockerFile.SaveToWriter(buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
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
