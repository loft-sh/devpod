package platform

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
)

func InitClientFromHost(ctx context.Context, devPodConfig *config.Config, devPodProHost string, log log.Logger) (client.Client, error) {
	provider, err := ProviderFromHost(ctx, devPodConfig, devPodProHost, log)
	if err != nil {
		return nil, fmt.Errorf("provider from pro instance: %w", err)
	}

	return InitClientFromProvider(ctx, devPodConfig, provider.Name, log)
}

func InitClientFromProvider(ctx context.Context, devPodConfig *config.Config, providerName string, log log.Logger) (client.Client, error) {
	configPath, err := LoftConfigPath(devPodConfig, providerName)
	if err != nil {
		return nil, fmt.Errorf("loft config path: %w", err)
	}

	return client.InitClientFromPath(ctx, configPath)
}

func ProviderFromHost(ctx context.Context, devPodConfig *config.Config, devPodProHost string, log log.Logger) (*provider.ProviderConfig, error) {
	proInstanceConfig, err := provider.LoadProInstanceConfig(devPodConfig.DefaultContext, devPodProHost)
	if err != nil {
		return nil, fmt.Errorf("load pro instance %s: %w", devPodProHost, err)
	}

	provider, err := workspace.FindProvider(devPodConfig, proInstanceConfig.Provider, log)
	if err != nil {
		return nil, fmt.Errorf("find provider: %w", err)
	} else if !provider.Config.IsProxyProvider() {
		return nil, fmt.Errorf("provider is not a proxy provider")
	}

	return provider.Config, nil
}

func LoftConfigPath(devPodConfig *config.Config, providerName string) (string, error) {
	providerDir, err := provider.GetProviderDir(devPodConfig.DefaultContext, providerName)
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(providerDir, "loft-config.json")

	return configPath, nil
}
