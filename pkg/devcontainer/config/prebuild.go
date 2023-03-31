package config

import (
	"encoding/json"
	"github.com/loft-sh/devpod/pkg/hash"
	"github.com/loft-sh/devpod/pkg/log"
)

func CalculatePrebuildHash(parsedConfig *DevContainerConfig, architecture, dockerfileContent string, log log.Logger) (string, error) {
	// TODO: is it a good idea to delete customizations before calculating the hash?
	parsedConfig = CloneDevContainerConfig(parsedConfig)
	parsedConfig.Customizations = nil

	// marshal the config
	configStr, err := json.Marshal(parsedConfig)
	if err != nil {
		return "", err
	}

	log.Debugf("Prebuild hash from: %s %s %s", architecture, string(configStr), dockerfileContent)
	return "devpod-" + hash.String(architecture + string(configStr) + dockerfileContent)[:32], nil
}
