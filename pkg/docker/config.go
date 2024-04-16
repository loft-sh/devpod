package docker

import (
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"

	"github.com/docker/docker/pkg/homedir"
)

const dockerFileFolder = ".docker"

func GetDockerConfigPath() string {
	configDir := os.Getenv("DOCKER_CONFIG")
	if configDir == "" {
		configDir = filepath.Join(homedir.Get(), dockerFileFolder)
	}

	return configDir
}

func LoadDockerConfig() (*configfile.ConfigFile, error) {
	return config.Load(GetDockerConfigPath())
}
