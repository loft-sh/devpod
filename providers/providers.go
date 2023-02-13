package providers

import (
	_ "embed"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/provider/providerimplementation"
	"github.com/pkg/errors"
	"strings"
)

//go:embed local/provider.yaml
var LocalProvider string

//go:embed gcloud/provider.yaml
var GCloudProvider string

//go:embed aws/provider.yaml
var AWSProvider string

// GetBuiltInProviders retrieves the built in providers
func GetBuiltInProviders(log log.Logger) (map[string]provider.Provider, error) {
	providers := []string{LocalProvider, GCloudProvider, AWSProvider}
	retProviders := map[string]provider.Provider{}

	// parse providers
	for _, providerConfig := range providers {
		parsedConfig, err := provider.ParseProvider(strings.NewReader(providerConfig))
		if err != nil {
			return nil, errors.Wrap(err, "parse provider")
		}

		retProviders[parsedConfig.Name] = providerimplementation.NewProvider(parsedConfig, log)
	}

	return retProviders, nil
}
