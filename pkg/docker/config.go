package docker

import (
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"os"
	"path/filepath"

	"github.com/docker/docker/pkg/homedir"
)

const dockerFileFolder = ".docker"

func LoadDockerConfig() (*configfile.ConfigFile, error) {
	configDir := os.Getenv("DOCKER_CONFIG")
	if configDir == "" {
		configDir = filepath.Join(homedir.Get(), dockerFileFolder)
	}

	return config.Load(configDir)
}
