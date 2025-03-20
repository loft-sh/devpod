package platform

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
)

func InitClientFromHost(ctx context.Context, devPodConfig *config.Config, devPodProHost string, log log.Logger) (client.Client, error) {
	provider, err := ProviderFromHost(ctx, devPodConfig, devPodProHost, log)
	if err != nil {
		return nil, fmt.Errorf("provider from pro instance: %w", err)
	}

	return InitClientFromProvider(ctx, devPodConfig, provider, log)
}

func InitClientFromProvider(ctx context.Context, devPodConfig *config.Config, providerName string, log log.Logger) (client.Client, error) {
	configPath, err := LoftConfigPath(devPodConfig.DefaultContext, providerName)
	if err != nil {
		return nil, fmt.Errorf("loft config path: %w", err)
	}

	return client.InitClientFromPath(ctx, configPath)
}

func ProviderFromHost(ctx context.Context, devPodConfig *config.Config, devPodProHost string, log log.Logger) (string, error) {
	proInstanceConfig, err := provider.LoadProInstanceConfig(devPodConfig.DefaultContext, devPodProHost)
	if err != nil {
		return "", fmt.Errorf("load pro instance %s: %w", devPodProHost, err)
	}

	return proInstanceConfig.Provider, nil
}

func LoftConfigPath(context string, providerName string) (string, error) {
	providerDir, err := provider.GetProviderDir(context, providerName)
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(providerDir, "loft-config.json")

	return configPath, nil
}
