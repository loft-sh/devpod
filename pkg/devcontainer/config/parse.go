package config

import (
	"bytes"
	"encoding/json"
	doublestar "github.com/bmatcuk/doublestar/v4"
	"github.com/loft-sh/devpod/pkg/scanner"
	"os"
	"path/filepath"
	"strings"
)

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
	return devContainer, nil
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
