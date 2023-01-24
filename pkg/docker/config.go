package docker

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"

	"github.com/docker/docker/pkg/homedir"
)

const dockerFileFolder = ".docker"

var configDir = os.Getenv("DOCKER_CONFIG")

var configDirOnce sync.Once

func LoadDockerConfig() (*configfile.ConfigFile, error) {
	configDirOnce.Do(func() {
		if configDir == "" {
			configDir = filepath.Join(homedir.Get(), dockerFileFolder)
		}
	})

	return config.Load(configDir)
}
