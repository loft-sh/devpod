package pro

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/loft"
	"github.com/loft-sh/devpod/pkg/version"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
)

// UpdateProvider currently only ensures the local provider is in sync with the remote for DevPod Pro instances
// Potentially auto-upgrade other providers in the future.
func UpdateProvider(devPodConfig *config.Config, providerName string, log log.Logger) error {
	proInstances, err := workspace.ListProInstances(devPodConfig, log)
	if err != nil {
		return fmt.Errorf("list pro instances: %w", err)
	} else if len(proInstances) == 0 {
		return nil
	}

	proInstance, ok := workspace.FindProviderProInstance(proInstances, providerName)
	if !ok {
		return nil
	}

	// compare versions
	newVersion, err := loft.GetProInstanceDevPodVersion(proInstance)
	if err != nil {
		return fmt.Errorf("version for pro instance %s: %w", proInstance.Host, err)
	}

	p, err := workspace.FindProvider(devPodConfig, proInstance.Provider, log)
	if err != nil {
		return fmt.Errorf("get provider config for pro provider %s: %w", proInstance.Provider, err)
	}
	if p.Config.Version == version.DevVersion {
		return nil
	}

	v1, err := semver.Parse(strings.TrimPrefix(newVersion, "v"))
	if err != nil {
		return fmt.Errorf("parse version %s: %w", newVersion, err)
	}
	v2, err := semver.Parse(strings.TrimPrefix(p.Config.Version, "v"))
	if err != nil {
		return fmt.Errorf("parse version %s: %w", p.Config.Version, err)
	}
	if v1.Compare(v2) == 0 {
		return nil
	}
	log.Infof("New provider version available, attempting to update %s", proInstance.Provider)

	providerSource, err := workspace.ResolveProviderSource(devPodConfig, proInstance.Provider, log)
	if err != nil {
		return fmt.Errorf("resolve provider source %s: %w", proInstance.Provider, err)
	}

	splitted := strings.Split(providerSource, "@")
	if len(splitted) == 0 {
		return fmt.Errorf("no provider source found %s", providerSource)
	}
	providerSource = splitted[0] + "@" + newVersion

	_, err = workspace.UpdateProvider(devPodConfig, providerName, providerSource, log)
	if err != nil {
		return fmt.Errorf("update provider %s: %w", proInstance.Provider, err)
	}

	log.Donef("Successfully updated provider %s", proInstance.Provider)
	return nil
}
