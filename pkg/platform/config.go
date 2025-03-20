package platform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/provider"
)

const (
	defaultTimeout                    = 10 * time.Minute
	LoftPlatformConfigFileName string = "loft-config.json" // TODO: replace hardcoded strings with this
)

func Timeout() time.Duration {
	if timeout := os.Getenv(TimeoutEnv); timeout != "" {
		if parsedTimeout, err := time.ParseDuration(timeout); err == nil {
			return parsedTimeout
		}
	}

	return defaultTimeout
}

// ReadConfig reads client.Config for given context and provider
func ReadConfig(contextName string, providerName string) (*client.Config, error) {
	// contextName is allowed to be empty
	providerDir, err := provider.GetProviderDir(contextName, providerName)
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(providerDir, LoftPlatformConfigFileName)

	// Check if given context and provider have Loft Platform configuration
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// If not just return empty response
		return nil, err
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	loftConfig := &client.Config{}
	err = json.Unmarshal(content, loftConfig)
	if err != nil {
		return nil, err
	}

	return loftConfig, nil
}
