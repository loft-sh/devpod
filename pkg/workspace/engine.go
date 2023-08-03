package workspace

import (
	"os"
	"strings"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/id"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
)

func ToEngineID(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.ToLower(url)
	url = workspaceIDRegEx2.ReplaceAllString(workspaceIDRegEx1.ReplaceAllString(url, "-"), "")
	url = strings.Trim(url, "-")
	return id.SafeConcatNameMax([]string{url}, 32)
}

func ListEngines(devPodConfig *config.Config, log log.Logger) ([]*provider2.Engine, error) {
	engineDir, err := provider2.GetEnginesDir(devPodConfig.DefaultContext)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(engineDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	retEngines := []*provider2.Engine{}
	for _, entry := range entries {
		engineConfig, err := provider2.LoadEngineConfig(devPodConfig.DefaultContext, entry.Name())
		if err != nil {
			log.ErrorStreamOnly().Warnf("Couldn't load engine %s: %v", entry.Name(), err)
			continue
		}

		retEngines = append(retEngines, engineConfig)
	}

	return retEngines, nil
}
