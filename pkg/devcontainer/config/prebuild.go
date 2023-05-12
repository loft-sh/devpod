package config

import (
	"encoding/json"
	"strings"

	"github.com/loft-sh/devpod/pkg/hash"
	"github.com/loft-sh/devpod/pkg/log"
)

func CalculatePrebuildHash(originalConfig *DevContainerConfig, platform, architecture, dockerfileContent string, log log.Logger) (string, error) {
	parsedConfig := CloneDevContainerConfig(originalConfig)

	if platform != "" {
		splitted := strings.Split(platform, "/")
		if len(splitted) == 2 && splitted[0] == "linux" {
			architecture = splitted[1]
		}
	}

	// delete all options that are not relevant for the build
	parsedConfig.Origin = ""
	parsedConfig.DevContainerActions = DevContainerActions{}
	parsedConfig.NonComposeBase = NonComposeBase{}
	parsedConfig.DevContainerConfigBase = DevContainerConfigBase{
		Name:                        parsedConfig.Name,
		Features:                    parsedConfig.Features,
		OverrideFeatureInstallOrder: parsedConfig.OverrideFeatureInstallOrder,
	}
	parsedConfig.ImageContainer = ImageContainer{
		Image: parsedConfig.Image,
	}
	parsedConfig.ComposeContainer = ComposeContainer{}
	parsedConfig.DockerfileContainer = DockerfileContainer{
		Dockerfile: parsedConfig.Dockerfile,
		Context:    parsedConfig.Context,
		Build:      parsedConfig.Build,
	}

	// marshal the config
	configStr, err := json.Marshal(parsedConfig)
	if err != nil {
		return "", err
	}

	log.Debugf("Prebuild hash from: %s %s %s", architecture, string(configStr), dockerfileContent)
	return "devpod-" + hash.String(architecture + string(configStr) + dockerfileContent)[:32], nil
}
