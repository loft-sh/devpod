package providers

import (
	_ "embed"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"strings"
)

//go:embed local/provider.yaml
var LocalProvider string

//go:embed ssh/provider.yaml
var SSHProvider string

// GetBuiltInProviders retrieves the built in providers
func GetBuiltInProviders() (map[string]*provider.ProviderConfig, error) {
	providers := []string{LocalProvider, SSHProvider}
	retProviderConfigs := map[string]*provider.ProviderConfig{}

	// parse providers
	for _, providerConfig := range providers {
		parsedConfig, err := provider.ParseProvider(strings.NewReader(providerConfig))
		if err != nil {
			return nil, errors.Wrap(err, "parse provider")
		}

		retProviderConfigs[parsedConfig.Name] = parsedConfig
	}

	return retProviderConfigs, nil
}
