package workspace

import (
	"os"
	"strings"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/id"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
)

func ToProInstanceID(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.ToLower(url)
	url = workspaceIDRegEx2.ReplaceAllString(workspaceIDRegEx1.ReplaceAllString(url, "-"), "")
	url = strings.Trim(url, "-")
	return id.SafeConcatNameMax([]string{url}, 32)
}

func ListProInstances(devPodConfig *config.Config, log log.Logger) ([]*provider2.ProInstance, error) {
	proInstanceDir, err := provider2.GetProInstancesDir(devPodConfig.DefaultContext)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(proInstanceDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	retProInstances := []*provider2.ProInstance{}
	for _, entry := range entries {
		proInstanceConfig, err := provider2.LoadProInstanceConfig(devPodConfig.DefaultContext, entry.Name())
		if err != nil {
			log.ErrorStreamOnly().Warnf("Couldn't load pro instance %s: %v", entry.Name(), err)
			continue
		}

		retProInstances = append(retProInstances, proInstanceConfig)
	}

	return retProInstances, nil
}
