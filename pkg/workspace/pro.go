package workspace

import (
	"os"

	"github.com/loft-sh/devpod/pkg/config"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
)

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

func FindProviderProInstance(proInstances []*provider2.ProInstance, providerName string) (*provider2.ProInstance, bool) {
	for _, instance := range proInstances {
		if instance.Provider == providerName {
			return instance, true
		}
	}

	return nil, false
}
