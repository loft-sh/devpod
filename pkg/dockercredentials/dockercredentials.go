package dockercredentials

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/config"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/devpod/pkg/file"
	"github.com/loft-sh/devpod/pkg/random"
	"github.com/pkg/errors"
)

type Request struct {
	// If ServerURL is empty its a list request
	ServerURL string
}

type ListResponse struct {
	Registries map[string]string
}

// Credentials holds the information shared between docker and the credentials store.
type Credentials struct {
	ServerURL string
	Username  string
	Secret    string
}

func ConfigureCredentialsContainer(userName string, port int) error {
	userHome, err := command.GetHome(userName)
	if err != nil {
		return err
	}

	configDir := os.Getenv("DOCKER_CONFIG")
	if configDir == "" {
		configDir = filepath.Join(userHome, ".docker")
	}

	return configureCredentials(userName, "/usr/local/bin", configDir, port)
}

func configureCredentials(userName string, targetDir, configDir string, port int) error {
	binaryPath, err := os.Executable()
	if err != nil {
		return err
	}

	err = file.MkdirAll(userName, configDir, 0777)
	if err != nil {
		return err
	}

	dockerConfig, err := config.Load(configDir)
	if err != nil {
		return err
	}

	// write credentials helper
	err = os.WriteFile(filepath.Join(targetDir, "docker-credential-devpod"), []byte(fmt.Sprintf(`#!/bin/sh
'%s' agent docker-credentials --port %d "$@"`, binaryPath, port)), 0777)
	if err != nil {
		return errors.Wrap(err, "write credential helper")
	}

	dockerConfig.CredentialsStore = "devpod"
	err = dockerConfig.Save()
	if err != nil {
		return err
	}

	err = file.Chown(userName, dockerConfig.Filename)
	if err != nil {
		return err
	}

	return nil
}

func ConfigureCredentialsMachine(targetFolder string, port int) (string, error) {
	dockerConfigDir := filepath.Join(targetFolder, ".cache", random.String(12))
	err := configureCredentials("", dockerConfigDir, dockerConfigDir, port)
	if err != nil {
		_ = os.RemoveAll(dockerConfigDir)
		return "", err
	}

	err = os.Setenv("DOCKER_CONFIG", dockerConfigDir)
	if err != nil {
		_ = os.RemoveAll(dockerConfigDir)
		return "", err
	}

	err = os.Setenv("PATH", os.Getenv("PATH")+":"+dockerConfigDir)
	if err != nil {
		_ = os.RemoveAll(dockerConfigDir)
		return "", err
	}

	return dockerConfigDir, nil
}

func ListCredentials() (*ListResponse, error) {
	dockerConfig, err := docker.LoadDockerConfig()
	if err != nil {
		return nil, err
	}

	allCredentials, err := dockerConfig.GetAllCredentials()
	if err != nil {
		return nil, err
	}

	retList := &ListResponse{Registries: map[string]string{}}
	for registryHostname, auth := range allCredentials {
		retList.Registries[registryHostname] = auth.Username
	}

	return retList, nil
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
