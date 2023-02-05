package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	doublestar "github.com/bmatcuk/doublestar/v4"
	"github.com/loft-sh/devpod/pkg/scanner"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"strings"
)

const DEVCONTAINER_FEATURE_FILE_NAME = "devcontainer-feature.json"

func ParseDevContainerFeature(folder string) (*FeatureConfig, error) {
	path := filepath.Join(folder, DEVCONTAINER_FEATURE_FILE_NAME)
	_, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("%s is missing in feature folder", DEVCONTAINER_FEATURE_FILE_NAME)
	}

	path, err = filepath.Abs(path)
	if err != nil {
		return nil, errors.Wrap(err, "make path absolute")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	featureConfig := &FeatureConfig{}
	err = json.Unmarshal(JSONCtoJSON(data), featureConfig)
	if err != nil {
		return nil, err
	}

	featureConfig.Origin = path
	return featureConfig, nil
}

func ParseDevContainerJSON(folder string) (*DevContainerConfig, error) {
	path := filepath.Join(folder, ".devcontainer", "devcontainer.json")
	_, err := os.Stat(path)
	if err != nil {
		path = filepath.Join(folder, ".devcontainer.json")
		_, err = os.Stat(path)
		if err != nil {
			matches, err := doublestar.FilepathGlob("./.devcontainer/**/devcontainer.json")
			if err != nil {
				return nil, err
			} else if len(matches) == 0 {
				return nil, nil
			}
		}
	}

	path, err = filepath.Abs(path)
	if err != nil {
		return nil, errors.Wrap(err, "make path absolute")
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	devContainer := &DevContainerConfig{}
	err = json.Unmarshal(JSONCtoJSON(bytes), devContainer)
	if err != nil {
		return nil, err
	}

	devContainer.Origin = path
	return replaceLegacy(devContainer)
}

func replaceLegacy(config *DevContainerConfig) (*DevContainerConfig, error) {
	if len(config.Extensions) == 0 && len(config.Settings) == 0 && config.DevPort == 0 {
		return config, nil
	}

	// make sure customizations exist
	if config.Customizations == nil {
		config.Customizations = map[string]interface{}{}
	}

	vsCodeConfig := &VSCodeCustomizations{}
	vscode, ok := config.Customizations["vscode"]
	if ok {
		err := Convert(vscode, &vsCodeConfig)
		if err != nil {
			return nil, err
		}
	}

	if len(config.Extensions) > 0 {
		vsCodeConfig.Extensions = config.Extensions
		config.Extensions = nil
	}

	if len(config.Settings) > 0 {
		if vsCodeConfig.Settings == nil {
			vsCodeConfig.Settings = map[string]interface{}{}
		}

		for k, v := range config.Settings {
			_, exists := vsCodeConfig.Settings[k]
			if !exists {
				vsCodeConfig.Settings[k] = v
			}
		}

		config.Settings = nil
	}

	if vsCodeConfig.DevPort == 0 {
		vsCodeConfig.DevPort = config.DevPort
		config.DevPort = 0
	}

	config.Customizations["vscode"] = vsCodeConfig
	return config, nil
}

func Convert(from interface{}, to interface{}) error {
	out, err := json.Marshal(from)
	if err != nil {
		return err
	}

	return json.Unmarshal(out, to)
}

func JSONCtoJSON(jsonCBytes []byte) []byte {
	scanner := scanner.NewScanner(bytes.NewReader(jsonCBytes))
	lines := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}

		lines = append(lines, line)
	}

	return []byte(strings.Join(lines, "\n"))
}
